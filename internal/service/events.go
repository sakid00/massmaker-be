package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/store"
)

const (
	maxEventBatch      = 40
	contactDedupWindow = 30 * time.Minute
)

type eventStore interface {
	InsertEvent(ctx context.Context, arg store.InsertEventParams) (store.Event, error)
	CountRecentContactClicks(ctx context.Context, arg store.CountRecentContactClicksParams) (int32, error)
	UpsertSession(ctx context.Context, arg store.UpsertSessionParams) error
	BackfillSessionUser(ctx context.Context, arg store.BackfillSessionUserParams) error
	ActivityTrackingGranted(ctx context.Context, arg store.ActivityTrackingGrantedParams) (bool, error)
}

type Events struct {
	repo eventStore
	now  func() time.Time
}

func NewEvents(db *store.DB) *Events {
	return &Events{repo: db.Queries, now: time.Now}
}

type IngestItem struct {
	Counted bool `json:"counted"`
}

func (s *Events) Ingest(ctx context.Context, raw json.RawMessage, userID *uuid.UUID, role string) ([]IngestItem, error) {
	if s.repo == nil {
		return nil, domain.ErrUnavailable
	}
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, domain.NewAppError(400, "validation", "invalid json")
	}
	if domain.HasPIIKeys(probe) {
		return nil, domain.NewAppError(400, "validation", "do not send phone, email, or whatsapp")
	}

	items, err := eventPayloads(raw)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, domain.NewAppError(400, "validation", "events must not be empty")
	}
	if len(items) > maxEventBatch {
		return nil, domain.NewAppError(400, "validation", "at most 40 events")
	}

	parsed := make([]store.InsertEventParams, 0, len(items))
	sessions := map[uuid.UUID]struct{}{}
	for _, item := range items {
		arg, err := parseEvent(item)
		if err != nil {
			return nil, err
		}
		if arg.Type == store.EventTypeContactClick && !domain.CanContactMakers(role) {
			return nil, domain.ErrForbidden
		}
		arg.EnmasseUserID = userID
		parsed = append(parsed, arg)
		sessions[arg.AnonSessionID] = struct{}{}
	}
	for sid := range sessions {
		ok, err := s.repo.ActivityTrackingGranted(ctx, store.ActivityTrackingGrantedParams{
			AnonSessionID: sid,
			UserID:        userID,
		})
		if err != nil {
			return nil, fmt.Errorf("activity consent: %w", err)
		}
		if !ok {
			return nil, domain.ErrConsentRequired
		}
	}

	type contactKey struct {
		session uuid.UUID
		maker   uuid.UUID
	}
	batchCounted := map[contactKey]int{}
	out := make([]IngestItem, 0, len(parsed))

	for _, arg := range parsed {
		if arg.Type == store.EventTypeContactClick && arg.MakerID != nil {
			key := contactKey{session: arg.AnonSessionID, maker: *arg.MakerID}
			if batchCounted[key] > 0 {
				arg.Counted = false
			} else {
				n, err := s.repo.CountRecentContactClicks(ctx, store.CountRecentContactClicksParams{
					AnonSessionID: arg.AnonSessionID,
					MakerID:       arg.MakerID,
					CreatedAt:     s.now().Add(-contactDedupWindow),
				})
				if err != nil {
					return nil, fmt.Errorf("count contact clicks: %w", err)
				}
				arg.Counted = n == 0
			}
			if arg.Counted {
				batchCounted[key]++
			}
		}

		if _, err := s.repo.InsertEvent(ctx, arg); err != nil {
			return nil, fmt.Errorf("ingest event: %w", err)
		}
		if err := s.repo.UpsertSession(ctx, store.UpsertSessionParams{
			AnonSessionID: arg.AnonSessionID,
			EnmasseUserID: userID,
		}); err != nil {
			return nil, fmt.Errorf("upsert session: %w", err)
		}
		sessions[arg.AnonSessionID] = struct{}{}
		out = append(out, IngestItem{Counted: arg.Counted})
	}

	if userID != nil {
		for sid := range sessions {
			if err := s.repo.BackfillSessionUser(ctx, store.BackfillSessionUserParams{
				AnonSessionID: sid,
				EnmasseUserID: userID,
			}); err != nil {
				return nil, fmt.Errorf("backfill session user: %w", err)
			}
		}
	}
	return out, nil
}

func eventPayloads(raw json.RawMessage) ([]json.RawMessage, error) {
	var batch struct {
		Events []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(raw, &batch); err != nil {
		return nil, domain.NewAppError(400, "validation", "invalid json")
	}
	if batch.Events != nil {
		return batch.Events, nil
	}
	return []json.RawMessage{raw}, nil
}

func parseEvent(raw json.RawMessage) (store.InsertEventParams, error) {
	var body struct {
		Type          string `json:"type"`
		TrafficSource string `json:"trafficSource"`
		AnonSessionID string `json:"anonSessionId"`
		Search        *struct {
			RawQuery            string `json:"rawQuery"`
			InterpretedCategory string `json:"interpretedCategory"`
			MatchedSynonym      string `json:"matchedSynonym"`
			ResultCount         int32  `json:"resultCount"`
			NoResultReason      string `json:"noResultReason"`
		} `json:"search"`
		ProfileView *struct {
			MakerID string `json:"makerId"`
		} `json:"profileView"`
		ContactClick *struct {
			MakerID string `json:"makerId"`
		} `json:"contactClick"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return store.InsertEventParams{}, domain.NewAppError(400, "validation", "invalid json")
	}

	sid, err := uuid.Parse(body.AnonSessionID)
	if err != nil {
		return store.InsertEventParams{}, domain.NewAppError(400, "validation", "anonSessionId must be a UUID")
	}
	src := body.TrafficSource
	if src == "" {
		src = "public"
	}
	switch src {
	case "public", "demo", "staff_test":
	default:
		return store.InsertEventParams{}, domain.NewAppError(400, "validation", "invalid trafficSource")
	}
	switch body.Type {
	case "search", "profile_view", "contact_click":
	default:
		return store.InsertEventParams{}, domain.NewAppError(400, "validation", "invalid type")
	}

	arg := store.InsertEventParams{
		Type:          store.EventType(body.Type),
		TrafficSource: store.TrafficSource(src),
		AnonSessionID: sid,
		Counted:       true,
	}
	if body.Search != nil {
		q := body.Search.RawQuery
		if len(q) > 120 {
			q = q[:120]
		}
		arg.RawQuery = strPtr(q)
		arg.InterpretedCategory = strPtr(body.Search.InterpretedCategory)
		arg.MatchedSynonym = strPtr(body.Search.MatchedSynonym)
		arg.ResultCount = &body.Search.ResultCount
		arg.NoResultReason = strPtr(body.Search.NoResultReason)
	}
	if body.ProfileView != nil {
		if id, err := uuid.Parse(body.ProfileView.MakerID); err == nil {
			arg.MakerID = &id
		}
	}
	if body.ContactClick != nil {
		if id, err := uuid.Parse(body.ContactClick.MakerID); err == nil {
			arg.MakerID = &id
		}
	}
	return arg, nil
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
