package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/enmasse"
	"github.com/sakid00/massmaker-be/internal/media"
	"github.com/sakid00/massmaker-be/internal/store"
)

type fakeProfile struct {
	complete bool
	profile  *enmasse.MeProfile
}

func (f *fakeProfile) ProfileStatus(context.Context, string) (enmasse.ProfileStatus, error) {
	return enmasse.ProfileStatus{Complete: f.complete, Missing: []string{}}, nil
}

func (f *fakeProfile) MeProfile(context.Context, string) (*enmasse.MeProfile, error) {
	return f.profile, nil
}

type fakeOrders struct {
	maker    store.GetMakerRow
	caps     []store.ListMakerCapabilitiesRow
	inserted store.Order
	atts     []store.OrderAttachment
}

func (f *fakeOrders) GetMaker(context.Context, uuid.UUID) (store.GetMakerRow, error) {
	if f.maker.ID == uuid.Nil {
		return store.GetMakerRow{}, pgx.ErrNoRows
	}
	return f.maker, nil
}
func (f *fakeOrders) ListMakerCapabilities(context.Context, []uuid.UUID) ([]store.ListMakerCapabilitiesRow, error) {
	return f.caps, nil
}
func (f *fakeOrders) GetMakerByEnmasseVendor(context.Context, *uuid.UUID) (store.GetMakerByEnmasseVendorRow, error) {
	return store.GetMakerByEnmasseVendorRow{}, pgx.ErrNoRows
}
func (f *fakeOrders) GetMakerByEnmasseUser(context.Context, *uuid.UUID) (store.GetMakerByEnmasseUserRow, error) {
	return store.GetMakerByEnmasseUserRow{}, pgx.ErrNoRows
}
func (f *fakeOrders) InsertOrder(_ context.Context, arg store.InsertOrderParams) (store.Order, error) {
	f.inserted = store.Order{
		ID:                uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ReferenceCode:     arg.ReferenceCode,
		MakerID:           arg.MakerID,
		EnmasseUserID:     arg.EnmasseUserID,
		ArtistDisplayName: arg.ArtistDisplayName,
		MakerName:         arg.MakerName,
		MakerCity:         arg.MakerCity,
		Status:            store.OrderStatusPendingReview,
		ProjectName:       arg.ProjectName,
		ProductName:       arg.ProductName,
		CategorySlug:      arg.CategorySlug,
		Quantity:          arg.Quantity,
		WantSample:        arg.WantSample,
		RequestedReadyOn:  arg.RequestedReadyOn,
		RequestedShipOn:   arg.RequestedShipOn,
		CreatedAt:         time.Date(2026, 9, 29, 4, 0, 0, 0, time.UTC),
	}
	return f.inserted, nil
}
func (f *fakeOrders) InsertOrderAttachment(_ context.Context, arg store.InsertOrderAttachmentParams) (store.OrderAttachment, error) {
	a := store.OrderAttachment{OrderID: arg.OrderID, Url: arg.Url, ContentType: arg.ContentType, OriginalFilename: arg.OriginalFilename, ByteSize: arg.ByteSize}
	f.atts = append(f.atts, a)
	return a, nil
}
func (f *fakeOrders) GetOrder(context.Context, uuid.UUID) (store.Order, error) {
	return f.inserted, nil
}
func (f *fakeOrders) ListOrdersByArtist(context.Context, uuid.UUID) ([]store.Order, error) {
	if f.inserted.ID == uuid.Nil {
		return []store.Order{}, nil
	}
	return []store.Order{f.inserted}, nil
}
func (f *fakeOrders) ListOrdersByMaker(context.Context, uuid.UUID) ([]store.Order, error) {
	return []store.Order{}, nil
}
func (f *fakeOrders) ListOrderAttachments(context.Context, uuid.UUID) ([]store.OrderAttachment, error) {
	return f.atts, nil
}
func (f *fakeOrders) ListOrderAttachmentsForOrders(context.Context, []uuid.UUID) ([]store.OrderAttachment, error) {
	return f.atts, nil
}
func (f *fakeOrders) UpdateOrderStatus(context.Context, store.UpdateOrderStatusParams) (store.Order, error) {
	return f.inserted, nil
}

func testArtistProfile() *enmasse.MeProfile {
	return &enmasse.MeProfile{
		Role:            "artist",
		ProfileComplete: true,
		Artist: &enmasse.MeArtist{
			ID:         "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
			ArtistName: "Studio K",
			Province:   "DKI Jakarta",
			City:       "Jakarta Selatan",
			District:   "Tebet",
			Address:    "Jl. Kenari 1",
		},
	}
}

func TestCreateInquiryForbiddenForVendor(t *testing.T) {
	t.Parallel()
	s := &Orders{
		q:       &fakeOrders{},
		enmasse: &fakeProfile{complete: true, profile: testArtistProfile()},
		now:     func() time.Time { return time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC) },
	}
	_, err := s.Create(context.Background(), "Bearer x", uuid.NewString(), "vendor", uuid.NewString(), CreateInquiryInput{})
	if err != domain.ErrForbidden {
		t.Fatalf("got %v", err)
	}
}

func TestCreateInquiryIncomplete(t *testing.T) {
	t.Parallel()
	s := &Orders{
		q:       &fakeOrders{},
		enmasse: &fakeProfile{complete: false, profile: testArtistProfile()},
		now:     time.Now,
	}
	_, err := s.Create(context.Background(), "Bearer x", uuid.NewString(), "artist", uuid.NewString(), CreateInquiryInput{})
	if err != domain.ErrProfileIncomplete {
		t.Fatalf("got %v", err)
	}
}

func TestCreateInquiryHappyPath(t *testing.T) {
	t.Parallel()
	makerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := &fakeOrders{
		maker: store.GetMakerRow{
			ID: makerID, Name: "Sablon Kita", City: "Bandung",
			PublicationStatus: store.PublicationStatusPublished,
		},
		caps: []store.ListMakerCapabilitiesRow{{MakerID: makerID, CategorySlug: "kaos"}},
	}
	s := &Orders{
		q:       repo,
		enmasse: &fakeProfile{complete: true, profile: testArtistProfile()},
		media:   media.ForPublicBase("https://cdn.example"),
		now:     func() time.Time { return time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC) },
	}
	got, err := s.Create(context.Background(), "Bearer x", userID.String(), "artist", makerID.String(), CreateInquiryInput{
		ProjectName:      "Booth JICAF",
		ProductName:      "Kaos event",
		CategorySlug:     "kaos",
		Quantity:         50,
		RequestedReadyOn: "2026-10-10",
		RequestedShipOn:  "2026-10-20",
		WantSample:       true,
		Attachments: []InquiryAttachmentIn{{
			URL:              "https://cdn.example/inquiries/" + userID.String() + "/design.jpg",
			ContentType:      "image/jpeg",
			OriginalFilename: "design.jpg",
			ByteSize:         1200,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.OrderPendingReview || got.ProjectName != "Booth JICAF" || got.ViewerRole != "artist" {
		t.Fatalf("got %+v", got)
	}
	if repo.inserted.MakerName != "Sablon Kita" || repo.inserted.ArtistDisplayName != "Studio K" {
		t.Fatalf("snapshots %+v", repo.inserted)
	}
}

func TestCreateInquiryRejectsOtherCategory(t *testing.T) {
	t.Parallel()
	makerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	s := &Orders{
		q: &fakeOrders{
			maker: store.GetMakerRow{ID: makerID, Name: "Sablon Kita", City: "Bandung", PublicationStatus: store.PublicationStatusPublished},
			caps:  []store.ListMakerCapabilitiesRow{{MakerID: makerID, CategorySlug: "kaos"}},
		},
		enmasse: &fakeProfile{complete: true, profile: testArtistProfile()},
		media:   media.ForPublicBase("https://cdn.example"),
		now:     func() time.Time { return time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC) },
	}
	_, err := s.Create(context.Background(), "Bearer x", userID.String(), "artist", makerID.String(), CreateInquiryInput{
		ProjectName:      "Booth",
		ProductName:      "Pin",
		CategorySlug:     "pin",
		Quantity:         10,
		RequestedReadyOn: "2026-10-10",
		RequestedShipOn:  "2026-10-20",
		Attachments: []InquiryAttachmentIn{{
			URL:         "https://cdn.example/inquiries/" + userID.String() + "/a.pdf",
			ContentType: "application/pdf",
			ByteSize:    100,
		}},
	})
	if err == nil {
		t.Fatal("expected category validation")
	}
}
