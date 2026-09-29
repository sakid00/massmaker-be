package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/store"
)

type fakeEvents struct {
	inserted     []store.InsertEventParams
	recent       int32
	sessions     []store.UpsertSessionParams
	backfills    []store.BackfillSessionUserParams
	denyActivity bool
}

func (f *fakeEvents) InsertEvent(_ context.Context, arg store.InsertEventParams) (store.Event, error) {
	f.inserted = append(f.inserted, arg)
	return store.Event{Counted: arg.Counted, EnmasseUserID: arg.EnmasseUserID, AnonSessionID: arg.AnonSessionID}, nil
}

func (f *fakeEvents) CountRecentContactClicks(context.Context, store.CountRecentContactClicksParams) (int32, error) {
	return f.recent, nil
}

func (f *fakeEvents) UpsertSession(_ context.Context, arg store.UpsertSessionParams) error {
	f.sessions = append(f.sessions, arg)
	return nil
}

func (f *fakeEvents) BackfillSessionUser(_ context.Context, arg store.BackfillSessionUserParams) error {
	f.backfills = append(f.backfills, arg)
	return nil
}

func (f *fakeEvents) ActivityTrackingGranted(context.Context, store.ActivityTrackingGrantedParams) (bool, error) {
	return !f.denyActivity, nil
}

func testEvents(repo *fakeEvents) *Events {
	return &Events{
		repo: repo,
		now:  func() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) },
	}
}

func TestIngestSingleSearch(t *testing.T) {
	t.Parallel()
	repo := &fakeEvents{}
	sid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	raw, _ := json.Marshal(map[string]any{
		"type":          "search",
		"anonSessionId": sid.String(),
		"search":        map[string]any{"rawQuery": "stiker", "interpretedCategory": "sticker", "resultCount": 3},
	})
	got, err := testEvents(repo).Ingest(context.Background(), raw, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Counted {
		t.Fatalf("got %+v", got)
	}
	if len(repo.inserted) != 1 || repo.inserted[0].Type != store.EventTypeSearch {
		t.Fatalf("inserted %+v", repo.inserted)
	}
	if repo.inserted[0].EnmasseUserID != nil {
		t.Fatal("anonymous flush must not stamp a user")
	}
}

func TestIngestBatch(t *testing.T) {
	t.Parallel()
	repo := &fakeEvents{}
	sid := "11111111-1111-1111-1111-111111111111"
	maker := "22222222-2222-2222-2222-222222222222"
	raw, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"type": "profile_view", "anonSessionId": sid, "profileView": map[string]any{"makerId": maker}},
			{"type": "contact_click", "anonSessionId": sid, "contactClick": map[string]any{"makerId": maker}},
		},
	})
	got, err := testEvents(repo).Ingest(context.Background(), raw, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
}

func TestIngestRejectsMoreThan40(t *testing.T) {
	t.Parallel()
	evs := make([]map[string]any, 41)
	for i := range evs {
		evs[i] = map[string]any{"type": "search", "anonSessionId": "11111111-1111-1111-1111-111111111111"}
	}
	raw, _ := json.Marshal(map[string]any{"events": evs})
	_, err := testEvents(&fakeEvents{}).Ingest(context.Background(), raw, nil, "")
	if err == nil {
		t.Fatal("expected validation")
	}
}

func TestIngestRejectsNestedPII(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]any{
		"type":          "search",
		"anonSessionId": "11111111-1111-1111-1111-111111111111",
		"search":        map[string]any{"email": "a@b.co"},
	})
	_, err := testEvents(&fakeEvents{}).Ingest(context.Background(), raw, nil, "")
	if err == nil {
		t.Fatal("expected pii reject")
	}
}

func TestContactClickDedupWindow(t *testing.T) {
	t.Parallel()
	repo := &fakeEvents{recent: 1}
	maker := "22222222-2222-2222-2222-222222222222"
	raw, _ := json.Marshal(map[string]any{
		"type":          "contact_click",
		"anonSessionId": "11111111-1111-1111-1111-111111111111",
		"contactClick":  map[string]any{"makerId": maker},
	})
	got, err := testEvents(repo).Ingest(context.Background(), raw, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Counted {
		t.Fatal("repeat contact in 30m must not count")
	}
}

func TestContactClickDedupInBatch(t *testing.T) {
	t.Parallel()
	repo := &fakeEvents{}
	sid := "11111111-1111-1111-1111-111111111111"
	maker := "22222222-2222-2222-2222-222222222222"
	raw, _ := json.Marshal(map[string]any{
		"events": []map[string]any{
			{"type": "contact_click", "anonSessionId": sid, "contactClick": map[string]any{"makerId": maker}},
			{"type": "contact_click", "anonSessionId": sid, "contactClick": map[string]any{"makerId": maker}},
		},
	})
	got, err := testEvents(repo).Ingest(context.Background(), raw, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !got[0].Counted || got[1].Counted {
		t.Fatalf("got %+v", got)
	}
}

func TestIngestRejectsVendorContactClick(t *testing.T) {
	t.Parallel()
	repo := &fakeEvents{}
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	raw, _ := json.Marshal(map[string]any{
		"type":          "contact_click",
		"anonSessionId": "11111111-1111-1111-1111-111111111111",
		"contactClick":  map[string]any{"makerId": "22222222-2222-2222-2222-222222222222"},
	})
	_, err := testEvents(repo).Ingest(context.Background(), raw, &uid, "vendor")
	if err != domain.ErrForbidden {
		t.Fatalf("got %v", err)
	}
	if len(repo.inserted) != 0 {
		t.Fatalf("vendor contact must not persist: %+v", repo.inserted)
	}
}

func TestIngestStampsUserAndBackfills(t *testing.T) {
	t.Parallel()
	repo := &fakeEvents{}
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	raw, _ := json.Marshal(map[string]any{
		"type":          "search",
		"anonSessionId": "11111111-1111-1111-1111-111111111111",
	})
	_, err := testEvents(repo).Ingest(context.Background(), raw, &uid, "artist")
	if err != nil {
		t.Fatal(err)
	}
	if repo.inserted[0].EnmasseUserID == nil || *repo.inserted[0].EnmasseUserID != uid {
		t.Fatal("expected user stamp")
	}
	if len(repo.backfills) != 1 || *repo.backfills[0].EnmasseUserID != uid {
		t.Fatalf("backfill %+v", repo.backfills)
	}
}

func TestIngestRejectsWithoutActivityGrant(t *testing.T) {
	t.Parallel()
	repo := &fakeEvents{denyActivity: true}
	raw, _ := json.Marshal(map[string]any{
		"type":          "search",
		"anonSessionId": "11111111-1111-1111-1111-111111111111",
	})
	_, err := testEvents(repo).Ingest(context.Background(), raw, nil, "")
	if err != domain.ErrConsentRequired {
		t.Fatalf("got %v", err)
	}
	if len(repo.inserted) != 0 {
		t.Fatalf("must not persist: %+v", repo.inserted)
	}
}
