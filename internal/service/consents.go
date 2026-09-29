package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/store"
)

type consentStore interface {
	GetConsentBySessionType(ctx context.Context, arg store.GetConsentBySessionTypeParams) (store.Consent, error)
	GetConsentByUserType(ctx context.Context, arg store.GetConsentByUserTypeParams) (store.Consent, error)
	ListConsentsByUser(ctx context.Context, userID *uuid.UUID) ([]store.Consent, error)
	InsertConsent(ctx context.Context, arg store.InsertConsentParams) (store.Consent, error)
	UpdateConsent(ctx context.Context, arg store.UpdateConsentParams) (store.Consent, error)
	BackfillConsentUser(ctx context.Context, arg store.BackfillConsentUserParams) error
}

type Consents struct {
	repo    consentStore
	now     func() time.Time
	pepper  string
	version string
}

func NewConsents(db *store.DB, pepper, version string) *Consents {
	if version == "" {
		version = "1.0"
	}
	return &Consents{repo: db.Queries, now: time.Now, pepper: pepper, version: version}
}

type PutConsentItem struct {
	ConsentType string
	Granted     bool
}

type PutConsentsInput struct {
	AnonSessionID string
	Source        string
	Items         []PutConsentItem
	UserID        *uuid.UUID
	Role          string
	IP            string
	UserAgent     string
}

type ConsentView struct {
	ConsentType string     `json:"consentType"`
	Granted     bool       `json:"granted"`
	GrantedAt   *time.Time `json:"grantedAt"`
	WithdrawnAt *time.Time `json:"withdrawnAt"`
	Source      string     `json:"source"`
	Version     string     `json:"version"`
}

func (s *Consents) Put(ctx context.Context, in PutConsentsInput) ([]ConsentView, error) {
	if s.repo == nil {
		return nil, domain.ErrUnavailable
	}
	sid, err := uuid.Parse(strings.TrimSpace(in.AnonSessionID))
	if err != nil {
		return nil, domain.NewAppError(400, "validation", "anonSessionId must be a UUID")
	}
	if !domain.ConsentSourceAllowed(in.Source) {
		return nil, domain.NewAppError(400, "validation", "invalid source")
	}
	if len(in.Items) == 0 {
		return nil, domain.NewAppError(400, "validation", "items must not be empty")
	}

	for _, item := range in.Items {
		if !domain.ConsentTypeAllowed(in.Role, item.ConsentType) {
			return nil, domain.ErrForbidden
		}
	}

	if in.UserID != nil {
		if err := s.repo.BackfillConsentUser(ctx, store.BackfillConsentUserParams{
			AnonSessionID: sid,
			UserID:        in.UserID,
		}); err != nil {
			return nil, fmt.Errorf("backfill consents: %w", err)
		}
	}

	now := s.now().UTC()
	ipHash := hashPeppered(s.pepper, in.IP)
	uaHash := hashPeppered(s.pepper, in.UserAgent)
	out := make([]ConsentView, 0, len(in.Items))
	for _, item := range in.Items {
		row, err := s.upsert(ctx, sid, in.UserID, in.Source, item, now, ipHash, uaHash)
		if err != nil {
			return nil, err
		}
		out = append(out, toConsentView(row))
	}
	return out, nil
}

func (s *Consents) ListMine(ctx context.Context, userID uuid.UUID) ([]ConsentView, error) {
	if s.repo == nil {
		return nil, domain.ErrUnavailable
	}
	uid := userID
	rows, err := s.repo.ListConsentsByUser(ctx, &uid)
	if err != nil {
		return nil, fmt.Errorf("list consents: %w", err)
	}
	out := make([]ConsentView, 0, len(rows))
	for _, row := range rows {
		out = append(out, toConsentView(row))
	}
	return out, nil
}

func (s *Consents) upsert(
	ctx context.Context,
	session uuid.UUID,
	userID *uuid.UUID,
	source string,
	item PutConsentItem,
	now time.Time,
	ipHash, uaHash string,
) (store.Consent, error) {
	typ := store.ConsentType(item.ConsentType)
	existing, err := s.find(ctx, session, userID, typ)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return store.Consent{}, fmt.Errorf("get consent: %w", err)
	}

	grantedAt, withdrawnAt := grantTimes(existing, err == nil, item.Granted, now)
	if errors.Is(err, pgx.ErrNoRows) {
		row, err := s.repo.InsertConsent(ctx, store.InsertConsentParams{
			UserID:        userID,
			AnonSessionID: session,
			ConsentType:   typ,
			Version:       s.version,
			Granted:       item.Granted,
			GrantedAt:     grantedAt,
			WithdrawnAt:   withdrawnAt,
			Source:        store.ConsentSource(source),
			IpHash:        ipHash,
			UserAgentHash: uaHash,
		})
		if err != nil {
			return store.Consent{}, fmt.Errorf("insert consent: %w", err)
		}
		return row, nil
	}

	row, err := s.repo.UpdateConsent(ctx, store.UpdateConsentParams{
		ID:            existing.ID,
		UserID:        userID,
		AnonSessionID: existing.AnonSessionID,
		Version:       s.version,
		Granted:       item.Granted,
		GrantedAt:     grantedAt,
		WithdrawnAt:   withdrawnAt,
		Source:        store.ConsentSource(source),
		IpHash:        ipHash,
		UserAgentHash: uaHash,
	})
	if err != nil {
		return store.Consent{}, fmt.Errorf("update consent: %w", err)
	}
	return row, nil
}

func (s *Consents) find(ctx context.Context, session uuid.UUID, userID *uuid.UUID, typ store.ConsentType) (store.Consent, error) {
	if userID != nil {
		row, err := s.repo.GetConsentByUserType(ctx, store.GetConsentByUserTypeParams{
			UserID:      userID,
			ConsentType: typ,
		})
		if err == nil || !errors.Is(err, pgx.ErrNoRows) {
			return row, err
		}
	}
	return s.repo.GetConsentBySessionType(ctx, store.GetConsentBySessionTypeParams{
		AnonSessionID: session,
		ConsentType:   typ,
	})
}

func grantTimes(existing store.Consent, found, granted bool, now time.Time) (pgtype.Timestamptz, pgtype.Timestamptz) {
	if granted {
		if !found || !existing.Granted || existing.WithdrawnAt.Valid {
			return timestamptz(now), pgtype.Timestamptz{}
		}
		return existing.GrantedAt, pgtype.Timestamptz{}
	}
	withdrawn := timestamptz(now)
	if found && !existing.Granted && existing.WithdrawnAt.Valid {
		withdrawn = existing.WithdrawnAt
	}
	grantedAt := pgtype.Timestamptz{}
	if found {
		grantedAt = existing.GrantedAt
	}
	return grantedAt, withdrawn
}

func toConsentView(row store.Consent) ConsentView {
	return ConsentView{
		ConsentType: string(row.ConsentType),
		Granted:     row.Granted,
		GrantedAt:   timePtr(row.GrantedAt),
		WithdrawnAt: timePtr(row.WithdrawnAt),
		Source:      string(row.Source),
		Version:     row.Version,
	}
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC()
	return &v
}

func hashPeppered(pepper, value string) string {
	mac := hmac.New(sha256.New, []byte(pepper))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
