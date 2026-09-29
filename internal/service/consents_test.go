package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/store"
)

type fakeConsents struct {
	byID      map[uuid.UUID]store.Consent
	backfills []store.BackfillConsentUserParams
}

func (f *fakeConsents) GetConsentBySessionType(_ context.Context, arg store.GetConsentBySessionTypeParams) (store.Consent, error) {
	for _, row := range f.byID {
		if row.AnonSessionID == arg.AnonSessionID && row.ConsentType == arg.ConsentType {
			return row, nil
		}
	}
	return store.Consent{}, pgx.ErrNoRows
}

func (f *fakeConsents) GetConsentByUserType(_ context.Context, arg store.GetConsentByUserTypeParams) (store.Consent, error) {
	if arg.UserID == nil {
		return store.Consent{}, pgx.ErrNoRows
	}
	for _, row := range f.byID {
		if row.UserID != nil && *row.UserID == *arg.UserID && row.ConsentType == arg.ConsentType {
			return row, nil
		}
	}
	return store.Consent{}, pgx.ErrNoRows
}

func (f *fakeConsents) ListConsentsByUser(_ context.Context, userID *uuid.UUID) ([]store.Consent, error) {
	if userID == nil {
		return nil, nil
	}
	var out []store.Consent
	for _, row := range f.byID {
		if row.UserID != nil && *row.UserID == *userID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeConsents) InsertConsent(_ context.Context, arg store.InsertConsentParams) (store.Consent, error) {
	row := store.Consent{
		ID:            uuid.New(),
		UserID:        arg.UserID,
		AnonSessionID: arg.AnonSessionID,
		ConsentType:   arg.ConsentType,
		Version:       arg.Version,
		Granted:       arg.Granted,
		GrantedAt:     arg.GrantedAt,
		WithdrawnAt:   arg.WithdrawnAt,
		Source:        arg.Source,
		IpHash:        arg.IpHash,
		UserAgentHash: arg.UserAgentHash,
		CreatedAt:     time.Date(2026, 9, 28, 5, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 9, 28, 5, 0, 0, 0, time.UTC),
	}
	if f.byID == nil {
		f.byID = map[uuid.UUID]store.Consent{}
	}
	f.byID[row.ID] = row
	return row, nil
}

func (f *fakeConsents) UpdateConsent(_ context.Context, arg store.UpdateConsentParams) (store.Consent, error) {
	row, ok := f.byID[arg.ID]
	if !ok {
		return store.Consent{}, pgx.ErrNoRows
	}
	if arg.UserID != nil {
		row.UserID = arg.UserID
	}
	row.AnonSessionID = arg.AnonSessionID
	row.Version = arg.Version
	row.Granted = arg.Granted
	row.GrantedAt = arg.GrantedAt
	row.WithdrawnAt = arg.WithdrawnAt
	row.Source = arg.Source
	row.IpHash = arg.IpHash
	row.UserAgentHash = arg.UserAgentHash
	row.UpdatedAt = time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC)
	f.byID[row.ID] = row
	return row, nil
}

func (f *fakeConsents) BackfillConsentUser(_ context.Context, arg store.BackfillConsentUserParams) error {
	f.backfills = append(f.backfills, arg)
	if arg.UserID == nil {
		return nil
	}
	occupied := map[store.ConsentType]struct{}{}
	for _, row := range f.byID {
		if row.UserID != nil && *row.UserID == *arg.UserID {
			occupied[row.ConsentType] = struct{}{}
		}
	}
	for id, row := range f.byID {
		if row.AnonSessionID != arg.AnonSessionID || row.UserID != nil {
			continue
		}
		if _, taken := occupied[row.ConsentType]; taken {
			continue
		}
		row.UserID = arg.UserID
		f.byID[id] = row
		occupied[row.ConsentType] = struct{}{}
	}
	return nil
}

func testConsents(repo *fakeConsents) *Consents {
	return &Consents{
		repo:    repo,
		now:     func() time.Time { return time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC) },
		pepper:  "test-pepper",
		version: "1.0",
	}
}

func guestPut(typ string, granted bool) PutConsentsInput {
	return PutConsentsInput{
		AnonSessionID: "11111111-1111-1111-1111-111111111111",
		Source:        domain.ConsentSourceActivityBanner,
		Items:         []PutConsentItem{{ConsentType: typ, Granted: granted}},
		IP:            "203.0.113.10",
		UserAgent:     "TestAgent/1.0",
	}
}

func TestPutGuestActivityGrant(t *testing.T) {
	t.Parallel()
	repo := &fakeConsents{}
	got, err := testConsents(repo).Put(context.Background(), guestPut(domain.ConsentActivityTracking, true))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Granted || got[0].GrantedAt == nil || got[0].WithdrawnAt != nil {
		t.Fatalf("got %+v", got)
	}
	if got[0].GrantedAt.Format(time.RFC3339) != "2026-09-28T06:00:00Z" {
		t.Fatalf("granted_at %v", got[0].GrantedAt)
	}
	if got[0].Version != "1.0" {
		t.Fatalf("version %s", got[0].Version)
	}
	var stored store.Consent
	for _, row := range repo.byID {
		stored = row
	}
	if stored.IpHash == "203.0.113.10" || stored.IpHash == "" {
		t.Fatalf("ip must be hashed, got %q", stored.IpHash)
	}
	if stored.UserAgentHash == "TestAgent/1.0" {
		t.Fatal("user agent must be hashed")
	}
}

func TestPutRejectsGuestDocumentTick(t *testing.T) {
	t.Parallel()
	_, err := testConsents(&fakeConsents{}).Put(context.Background(), guestPut(domain.ConsentPrivacyPolicy, true))
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestPutRejectsArtistWhatsApp(t *testing.T) {
	t.Parallel()
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	in := guestPut(domain.ConsentWhatsAppPublication, true)
	in.UserID = &uid
	in.Role = "artist"
	in.Source = domain.ConsentSourceOnboarding
	_, err := testConsents(&fakeConsents{}).Put(context.Background(), in)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestPutIgnoresClientTimestamps(t *testing.T) {
	t.Parallel()
	got, err := testConsents(&fakeConsents{}).Put(context.Background(), guestPut(domain.ConsentMarketing, true))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].GrantedAt == nil || got[0].GrantedAt.Format(time.RFC3339) != "2026-09-28T06:00:00Z" {
		t.Fatalf("server clock: %+v", got[0].GrantedAt)
	}
}

func TestPutUpsertKeepsGrantedAtUntilWithdraw(t *testing.T) {
	t.Parallel()
	repo := &fakeConsents{}
	svc := testConsents(repo)
	first, err := svc.Put(context.Background(), guestPut(domain.ConsentActivityTracking, true))
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 9, 28, 7, 0, 0, 0, time.UTC) }
	second, err := svc.Put(context.Background(), guestPut(domain.ConsentActivityTracking, true))
	if err != nil {
		t.Fatal(err)
	}
	if !second[0].GrantedAt.Equal(*first[0].GrantedAt) {
		t.Fatalf("re-save must keep granted_at: %v vs %v", second[0].GrantedAt, first[0].GrantedAt)
	}
	if len(repo.byID) != 1 {
		t.Fatalf("upsert must keep one row, got %d", len(repo.byID))
	}
	withdraw, err := svc.Put(context.Background(), guestPut(domain.ConsentActivityTracking, false))
	if err != nil {
		t.Fatal(err)
	}
	if withdraw[0].Granted || withdraw[0].WithdrawnAt == nil {
		t.Fatalf("withdraw %+v", withdraw[0])
	}
	if withdraw[0].WithdrawnAt.Format(time.RFC3339) != "2026-09-28T07:00:00Z" {
		t.Fatalf("withdrawn_at %v", withdraw[0].WithdrawnAt)
	}
	regrant, err := svc.Put(context.Background(), guestPut(domain.ConsentActivityTracking, true))
	if err != nil {
		t.Fatal(err)
	}
	if regrant[0].GrantedAt.Format(time.RFC3339) != "2026-09-28T07:00:00Z" || regrant[0].WithdrawnAt != nil {
		t.Fatalf("re-grant %+v", regrant[0])
	}
}

func TestPutGuestThenLoginBackfillsUser(t *testing.T) {
	t.Parallel()
	repo := &fakeConsents{}
	svc := testConsents(repo)
	if _, err := svc.Put(context.Background(), guestPut(domain.ConsentActivityTracking, true)); err != nil {
		t.Fatal(err)
	}
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	in := guestPut(domain.ConsentMarketing, true)
	in.UserID = &uid
	in.Role = "artist"
	got, err := svc.Put(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	if len(repo.backfills) != 1 || *repo.backfills[0].UserID != uid {
		t.Fatalf("backfill %+v", repo.backfills)
	}
	listed, err := svc.ListMine(context.Background(), uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("listed %+v", listed)
	}
}

func TestPutRejectsUnknownSource(t *testing.T) {
	t.Parallel()
	in := guestPut(domain.ConsentActivityTracking, true)
	in.Source = "admin_override"
	_, err := testConsents(&fakeConsents{}).Put(context.Background(), in)
	if err == nil {
		t.Fatal("expected validation")
	}
}

func TestGrantTimesNilWhenNeverGranted(t *testing.T) {
	t.Parallel()
	gotAt, withdrawn := grantTimes(store.Consent{}, false, false, time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC))
	if gotAt.Valid {
		t.Fatal("never granted has no granted_at")
	}
	if !withdrawn.Valid {
		t.Fatal("withdraw stamps withdrawn_at")
	}
}

func TestHashPepperedStable(t *testing.T) {
	t.Parallel()
	a := hashPeppered("pepper", "1.2.3.4")
	b := hashPeppered("pepper", "1.2.3.4")
	c := hashPeppered("other", "1.2.3.4")
	if a != b || a == "1.2.3.4" || a == c {
		t.Fatalf("a=%s b=%s c=%s", a, b, c)
	}
}
