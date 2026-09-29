package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/enmasse"
	"github.com/sakid00/massmaker-be/internal/media"
	"github.com/sakid00/massmaker-be/internal/store"
)

type profileClient interface {
	ProfileStatus(ctx context.Context, userID string) (enmasse.ProfileStatus, error)
	MeProfile(ctx context.Context, authorization string) (*enmasse.MeProfile, error)
}

type orderQueries interface {
	GetMaker(ctx context.Context, id uuid.UUID) (store.GetMakerRow, error)
	ListMakerCapabilities(ctx context.Context, makerIds []uuid.UUID) ([]store.ListMakerCapabilitiesRow, error)
	GetMakerByEnmasseVendor(ctx context.Context, enmasseVendorID *uuid.UUID) (store.GetMakerByEnmasseVendorRow, error)
	GetMakerByEnmasseUser(ctx context.Context, enmasseUserID *uuid.UUID) (store.GetMakerByEnmasseUserRow, error)
	InsertOrder(ctx context.Context, arg store.InsertOrderParams) (store.Order, error)
	InsertOrderAttachment(ctx context.Context, arg store.InsertOrderAttachmentParams) (store.OrderAttachment, error)
	GetOrder(ctx context.Context, id uuid.UUID) (store.Order, error)
	ListOrdersByArtist(ctx context.Context, enmasseUserID uuid.UUID) ([]store.Order, error)
	ListOrdersByMaker(ctx context.Context, makerID uuid.UUID) ([]store.Order, error)
	ListOrderAttachments(ctx context.Context, orderID uuid.UUID) ([]store.OrderAttachment, error)
	ListOrderAttachmentsForOrders(ctx context.Context, orderIds []uuid.UUID) ([]store.OrderAttachment, error)
	UpdateOrderStatus(ctx context.Context, arg store.UpdateOrderStatusParams) (store.Order, error)
}

type Orders struct {
	q       orderQueries
	pool    *pgxpool.Pool
	enmasse profileClient
	media   *media.Presigner
	now     func() time.Time
}

func NewOrders(db *store.DB, em *enmasse.Client, presigner *media.Presigner) *Orders {
	s := &Orders{enmasse: em, media: presigner, now: time.Now}
	if db != nil {
		s.q = db.Queries
		s.pool = db.Pool
	}
	return s
}

type InquiryAttachmentIn struct {
	URL              string `json:"url"`
	ContentType      string `json:"contentType"`
	OriginalFilename string `json:"originalFilename"`
	ByteSize         int32  `json:"byteSize"`
}

type CreateInquiryInput struct {
	ProjectName      string                `json:"projectName"`
	ProductName      string                `json:"productName"`
	CategorySlug     string                `json:"categorySlug"`
	Quantity         int32                 `json:"quantity"`
	QuantityNote     string                `json:"quantityNote"`
	DesignCount      int32                 `json:"designCount"`
	WantSample       bool                  `json:"wantSample"`
	SampleQuantity   *int32                `json:"sampleQuantity"`
	RequestedReadyOn string                `json:"requestedReadyOn"`
	RequestedShipOn  string                `json:"requestedShipOn"`
	Rush             bool                  `json:"rush"`
	BudgetMinIDR     *int32                `json:"budgetMinIdr"`
	BudgetMaxIDR     *int32                `json:"budgetMaxIdr"`
	Notes            string                `json:"notes"`
	Attachments      []InquiryAttachmentIn `json:"attachments"`
}

type PatchOrderInput struct {
	Status     string `json:"status"`
	VendorNote string `json:"vendorNote"`
}

type OrderAttachmentPublic struct {
	URL              string `json:"url"`
	ContentType      string `json:"contentType"`
	OriginalFilename string `json:"originalFilename"`
	ByteSize         int32  `json:"byteSize"`
}

type OrderCard struct {
	ID               string `json:"id"`
	ReferenceCode    string `json:"referenceCode"`
	Status           string `json:"status"`
	ProjectName      string `json:"projectName"`
	ProductName      string `json:"productName"`
	CategorySlug     string `json:"categorySlug"`
	WantSample       bool   `json:"wantSample"`
	Quantity         int32  `json:"quantity"`
	RequestedShipOn  string `json:"requestedShipOn"`
	CreatedAt        string `json:"createdAt"`
	CounterpartyName string `json:"counterpartyName"`
	CounterpartyCity string `json:"counterpartyCity"`
	ViewerRole       string `json:"viewerRole"`
	PrimaryImage     *Media `json:"primaryImage"`
}

type OrderDetail struct {
	OrderCard
	MakerID          string                  `json:"makerId"`
	QuantityNote     *string                 `json:"quantityNote"`
	DesignCount      int32                   `json:"designCount"`
	SampleQuantity   *int32                  `json:"sampleQuantity"`
	RequestedReadyOn string                  `json:"requestedReadyOn"`
	Rush             bool                    `json:"rush"`
	BudgetMinIDR     *int32                  `json:"budgetMinIdr"`
	BudgetMaxIDR     *int32                  `json:"budgetMaxIdr"`
	Notes            *string                 `json:"notes"`
	ShipToProvince   string                  `json:"shipToProvince"`
	ShipToCity       string                  `json:"shipToCity"`
	ShipToDistrict   string                  `json:"shipToDistrict"`
	ShipToAddress    *string                 `json:"shipToAddress"`
	VendorNote       *string                 `json:"vendorNote"`
	Attachments      []OrderAttachmentPublic `json:"attachments"`
}

type InquiryArtistContext struct {
	DisplayName string `json:"displayName"`
	Province    string `json:"province"`
	City        string `json:"city"`
	District    string `json:"district"`
}

type InquiryContext struct {
	Maker  MakerDetail          `json:"maker"`
	Artist InquiryArtistContext `json:"artist"`
}

func (s *Orders) requireArtistComplete(ctx context.Context, userID, role string) error {
	if s.enmasse == nil {
		return domain.ErrUnavailable
	}
	if !domain.CanCreateInquiry(role) {
		return domain.ErrForbidden
	}
	st, err := s.enmasse.ProfileStatus(ctx, userID)
	if err != nil {
		return err
	}
	if !st.Complete {
		return domain.ErrProfileIncomplete
	}
	return nil
}

func (s *Orders) Presign(ctx context.Context, userID, role, contentType string, contentLength int64) (*media.Result, error) {
	if err := s.requireArtistComplete(ctx, userID, role); err != nil {
		return nil, err
	}
	if s.media == nil {
		return nil, domain.ErrMediaUnavailable
	}
	return s.media.PresignInquiry(ctx, userID, contentType, contentLength)
}

func (s *Orders) Context(ctx context.Context, authorization, userID, role, makerID string) (*InquiryContext, int, error) {
	if err := s.requireArtistComplete(ctx, userID, role); err != nil {
		return nil, 0, err
	}
	profile, err := s.enmasse.MeProfile(ctx, authorization)
	if err != nil {
		return nil, 0, err
	}
	artist, err := artistSnapshot(profile)
	if err != nil {
		return nil, 0, err
	}
	id, err := uuid.Parse(makerID)
	if err != nil {
		return nil, 0, domain.ErrNotFound
	}
	detail, unavail, code, err := s.makerDetail(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	if unavail != nil {
		return nil, code, nil
	}
	return &InquiryContext{Maker: *detail, Artist: artist}, http.StatusOK, nil
}

func (s *Orders) Create(ctx context.Context, authorization, userID, role, makerID string, in CreateInquiryInput) (*OrderDetail, error) {
	if s.q == nil {
		return nil, domain.ErrUnavailable
	}
	if err := s.requireArtistComplete(ctx, userID, role); err != nil {
		return nil, err
	}
	profile, err := s.enmasse.MeProfile(ctx, authorization)
	if err != nil {
		return nil, err
	}
	artist, err := artistSnapshot(profile)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	mid, err := uuid.Parse(makerID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	row, err := s.q.GetMaker(ctx, mid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("order maker: %w", err)
	}
	code := domain.PublicMakerCode(string(row.PublicationStatus), row.IsDemoFixture)
	if code == http.StatusGone {
		return nil, domain.ErrGone
	}
	if code != http.StatusOK {
		return nil, domain.ErrNotFound
	}
	caps, err := s.q.ListMakerCapabilities(ctx, []uuid.UUID{mid})
	if err != nil {
		return nil, fmt.Errorf("order caps: %w", err)
	}
	arg, atts, err := s.validateCreate(uid, row, caps, artist, profile, in)
	if err != nil {
		return nil, err
	}

	q := s.q
	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("order begin: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		q = store.New(tx)
		order, err := q.InsertOrder(ctx, arg)
		if err != nil {
			return nil, fmt.Errorf("order insert: %w", err)
		}
		saved, err := insertAttachments(ctx, q, order.ID, atts)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("order commit: %w", err)
		}
		detail := toOrderDetail(order, saved, "artist")
		return &detail, nil
	}

	order, err := q.InsertOrder(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("order insert: %w", err)
	}
	saved, err := insertAttachments(ctx, q, order.ID, atts)
	if err != nil {
		return nil, err
	}
	detail := toOrderDetail(order, saved, "artist")
	return &detail, nil
}

func (s *Orders) List(ctx context.Context, authorization, userID, role string) ([]OrderCard, error) {
	if s.q == nil {
		return nil, domain.ErrUnavailable
	}
	if !domain.CanListOrders(role) {
		return nil, domain.ErrForbidden
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	var rows []store.Order
	if role == "artist" {
		rows, err = s.q.ListOrdersByArtist(ctx, uid)
	} else {
		var makerID uuid.UUID
		var ok bool
		makerID, ok, err = s.vendorMakerID(ctx, authorization, uid)
		if err != nil {
			return nil, err
		}
		if !ok {
			return []OrderCard{}, nil
		}
		rows, err = s.q.ListOrdersByMaker(ctx, makerID)
	}
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	var atts []store.OrderAttachment
	if len(ids) > 0 {
		atts, err = s.q.ListOrderAttachmentsForOrders(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("list order attachments: %w", err)
		}
	}
	byOrder := map[uuid.UUID][]store.OrderAttachment{}
	for _, a := range atts {
		byOrder[a.OrderID] = append(byOrder[a.OrderID], a)
	}
	out := make([]OrderCard, 0, len(rows))
	for _, row := range rows {
		out = append(out, toOrderCard(row, byOrder[row.ID], role))
	}
	return out, nil
}

func (s *Orders) Get(ctx context.Context, authorization, userID, role, orderID string) (*OrderDetail, error) {
	order, atts, err := s.loadOwned(ctx, authorization, userID, role, orderID)
	if err != nil {
		return nil, err
	}
	detail := toOrderDetail(order, atts, role)
	return &detail, nil
}

func (s *Orders) Patch(ctx context.Context, authorization, userID, role, orderID string, in PatchOrderInput) (*OrderDetail, error) {
	order, _, err := s.loadOwned(ctx, authorization, userID, role, orderID)
	if err != nil {
		return nil, err
	}
	if err := domain.ApplyOrderStatus(role, string(order.Status), in.Status); err != nil {
		return nil, err
	}
	note := order.VendorNote
	if role == "vendor" {
		n, err := domain.ClampOrderNote(in.VendorNote)
		if err != nil {
			return nil, err
		}
		note = strPtr(n)
	}
	updated, err := s.q.UpdateOrderStatus(ctx, store.UpdateOrderStatusParams{
		ID:         order.ID,
		Status:     store.OrderStatus(strings.TrimSpace(in.Status)),
		VendorNote: note,
	})
	if err != nil {
		return nil, fmt.Errorf("patch order: %w", err)
	}
	atts, err := s.q.ListOrderAttachments(ctx, updated.ID)
	if err != nil {
		return nil, fmt.Errorf("patch order attachments: %w", err)
	}
	detail := toOrderDetail(updated, atts, role)
	return &detail, nil
}

func (s *Orders) loadOwned(ctx context.Context, authorization, userID, role, orderID string) (store.Order, []store.OrderAttachment, error) {
	if s.q == nil {
		return store.Order{}, nil, domain.ErrUnavailable
	}
	if !domain.CanListOrders(role) {
		return store.Order{}, nil, domain.ErrForbidden
	}
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return store.Order{}, nil, domain.ErrNotFound
	}
	order, err := s.q.GetOrder(ctx, oid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Order{}, nil, domain.ErrNotFound
		}
		return store.Order{}, nil, fmt.Errorf("get order: %w", err)
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return store.Order{}, nil, domain.ErrUnauthorized
	}
	if role == "artist" {
		if order.EnmasseUserID != uid {
			return store.Order{}, nil, domain.ErrForbidden
		}
	} else {
		makerID, ok, err := s.vendorMakerID(ctx, authorization, uid)
		if err != nil {
			return store.Order{}, nil, err
		}
		if !ok || order.MakerID != makerID {
			return store.Order{}, nil, domain.ErrForbidden
		}
	}
	atts, err := s.q.ListOrderAttachments(ctx, order.ID)
	if err != nil {
		return store.Order{}, nil, fmt.Errorf("order attachments: %w", err)
	}
	return order, atts, nil
}

func (s *Orders) vendorMakerID(ctx context.Context, authorization string, userID uuid.UUID) (uuid.UUID, bool, error) {
	if s.enmasse == nil {
		return uuid.Nil, false, domain.ErrUnavailable
	}
	profile, err := s.enmasse.MeProfile(ctx, authorization)
	if err != nil {
		return uuid.Nil, false, err
	}
	if profile.Vendor != nil && strings.TrimSpace(profile.Vendor.ID) != "" {
		vid, err := uuid.Parse(profile.Vendor.ID)
		if err == nil {
			row, err := s.q.GetMakerByEnmasseVendor(ctx, &vid)
			if err == nil {
				return row.ID, true, nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return uuid.Nil, false, fmt.Errorf("vendor maker: %w", err)
			}
		}
	}
	row, err := s.q.GetMakerByEnmasseUser(ctx, &userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, fmt.Errorf("vendor maker by user: %w", err)
	}
	return row.ID, true, nil
}

func (s *Orders) makerDetail(ctx context.Context, id uuid.UUID) (*MakerDetail, *MakerUnavailable, int, error) {
	row, err := s.q.GetMaker(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unavailable(nil, "not_found"), http.StatusNotFound, nil
		}
		return nil, nil, 0, fmt.Errorf("inquiry maker: %w", err)
	}
	code := domain.PublicMakerCode(string(row.PublicationStatus), row.IsDemoFixture)
	sid := id.String()
	if code == http.StatusGone {
		return nil, unavailable(&sid, "withdrawn"), code, nil
	}
	if code != http.StatusOK {
		return nil, unavailable(&sid, "not_found"), code, nil
	}
	caps, err := s.q.ListMakerCapabilities(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("inquiry caps: %w", err)
	}
	vdate := dateString(row.VerificationDate)
	now := s.now()
	return &MakerDetail{
		ID:                  sid,
		Name:                row.Name,
		City:                row.City,
		Capabilities:        toCaps(caps),
		MOQ:                 toMOQ(row.MoqQuantity, moqBasis(row.MoqBasis), row.MoqBasisNote),
		LeadTimeEstimate:    row.LeadTimeEstimate,
		PriceRange:          toPrice(row.PriceMinIdr, row.PriceMaxIdr, row.PriceNote),
		Conditions:          row.Conditions,
		VerificationDate:    vdate,
		NeedsReconfirmation: domain.NeedsReconfirmation(vdate, now.Year(), int(now.Month()), now.Day()),
		Images:              []Media{},
		Recommendations:     []Recommendation{},
		Badges:              []any{},
		Contact:             nil,
	}, nil, http.StatusOK, nil
}

func (s *Orders) validateCreate(uid uuid.UUID, maker store.GetMakerRow, caps []store.ListMakerCapabilitiesRow, artist InquiryArtistContext, profile *enmasse.MeProfile, in CreateInquiryInput) (store.InsertOrderParams, []store.InsertOrderAttachmentParams, error) {
	project, err := domain.ClampOrderName(in.ProjectName, "projectName")
	if err != nil {
		return store.InsertOrderParams{}, nil, err
	}
	product, err := domain.ClampOrderName(in.ProductName, "productName")
	if err != nil {
		return store.InsertOrderParams{}, nil, err
	}
	slug := strings.TrimSpace(in.CategorySlug)
	allowed := false
	for _, c := range caps {
		if c.CategorySlug == slug {
			allowed = true
			break
		}
	}
	if !allowed {
		return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "category is not offered by this maker")
	}
	if in.Quantity <= 0 {
		return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "quantity must be positive")
	}
	designCount := in.DesignCount
	if designCount == 0 {
		designCount = 1
	}
	if designCount < 1 {
		return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "designCount must be positive")
	}
	qtyNote, err := domain.ClampOrderNote(in.QuantityNote)
	if err != nil {
		return store.InsertOrderParams{}, nil, err
	}
	notes, err := domain.ClampOrderNote(in.Notes)
	if err != nil {
		return store.InsertOrderParams{}, nil, err
	}
	ready, err := domain.ParseOrderDate(in.RequestedReadyOn)
	if err != nil {
		return store.InsertOrderParams{}, nil, err
	}
	ship, err := domain.ParseOrderDate(in.RequestedShipOn)
	if err != nil {
		return store.InsertOrderParams{}, nil, err
	}
	if err := domain.ValidateOrderDates(ready, ship, s.now()); err != nil {
		return store.InsertOrderParams{}, nil, err
	}
	if in.BudgetMinIDR != nil && *in.BudgetMinIDR < 0 {
		return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "invalid budget")
	}
	if in.BudgetMaxIDR != nil && *in.BudgetMaxIDR < 0 {
		return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "invalid budget")
	}
	if in.BudgetMinIDR != nil && in.BudgetMaxIDR != nil && *in.BudgetMaxIDR < *in.BudgetMinIDR {
		return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "invalid budget")
	}
	sampleQty := in.SampleQuantity
	if in.WantSample {
		if sampleQty == nil {
			one := int32(1)
			sampleQty = &one
		} else if *sampleQty <= 0 {
			return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "sampleQuantity must be positive")
		}
	} else {
		sampleQty = nil
	}
	if len(in.Attachments) < 1 || len(in.Attachments) > domain.MaxOrderAttachments {
		return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "upload one to eight design files")
	}
	base := ""
	if s.media != nil {
		base = s.media.PublicBaseURL()
	}
	atts := make([]store.InsertOrderAttachmentParams, 0, len(in.Attachments))
	for i, a := range in.Attachments {
		if _, err := media.InquiryExtForContentType(a.ContentType); err != nil {
			return store.InsertOrderParams{}, nil, err
		}
		if err := media.ValidateInquiryContentLength(a.ContentType, int64(a.ByteSize)); err != nil {
			return store.InsertOrderParams{}, nil, err
		}
		if !media.InquiryURLOwned(a.URL, base, uid.String()) {
			return store.InsertOrderParams{}, nil, domain.NewAppError(http.StatusBadRequest, "validation", "attachment url is invalid")
		}
		name := strings.TrimSpace(a.OriginalFilename)
		if name == "" {
			name = fmt.Sprintf("file-%d", i+1)
		}
		atts = append(atts, store.InsertOrderAttachmentParams{
			Url:              strings.TrimSpace(a.URL),
			ContentType:      strings.ToLower(strings.TrimSpace(a.ContentType)),
			OriginalFilename: name,
			ByteSize:         a.ByteSize,
			SortOrder:        int32(i),
		})
	}

	id := uuid.New()
	var artistID *uuid.UUID
	if profile.Artist != nil {
		if parsed, err := uuid.Parse(profile.Artist.ID); err == nil {
			artistID = &parsed
		}
	}
	var address *string
	if profile.Artist != nil {
		address = strPtr(strings.TrimSpace(profile.Artist.Address))
	}
	return store.InsertOrderParams{
		ReferenceCode:     referenceCode(s.now(), id),
		MakerID:           maker.ID,
		EnmasseUserID:     uid,
		EnmasseArtistID:   artistID,
		ArtistDisplayName: artist.DisplayName,
		MakerName:         maker.Name,
		MakerCity:         maker.City,
		ProjectName:       project,
		ProductName:       product,
		CategorySlug:      slug,
		Quantity:          in.Quantity,
		QuantityNote:      strPtr(qtyNote),
		DesignCount:       designCount,
		WantSample:        in.WantSample,
		SampleQuantity:    sampleQty,
		RequestedReadyOn:  ready,
		RequestedShipOn:   ship,
		Rush:              in.Rush,
		BudgetMinIdr:      in.BudgetMinIDR,
		BudgetMaxIdr:      in.BudgetMaxIDR,
		Notes:             strPtr(notes),
		ShipToProvince:    artist.Province,
		ShipToCity:        artist.City,
		ShipToDistrict:    artist.District,
		ShipToAddress:     address,
	}, atts, nil
}

func insertAttachments(ctx context.Context, q orderQueries, orderID uuid.UUID, atts []store.InsertOrderAttachmentParams) ([]store.OrderAttachment, error) {
	saved := make([]store.OrderAttachment, 0, len(atts))
	for _, a := range atts {
		a.OrderID = orderID
		row, err := q.InsertOrderAttachment(ctx, a)
		if err != nil {
			return nil, fmt.Errorf("order attachment: %w", err)
		}
		saved = append(saved, row)
	}
	return saved, nil
}

func artistSnapshot(profile *enmasse.MeProfile) (InquiryArtistContext, error) {
	if profile == nil || profile.Artist == nil {
		return InquiryArtistContext{}, domain.ErrProfileIncomplete
	}
	a := profile.Artist
	name := strings.TrimSpace(a.ArtistName)
	if name == "" {
		name = strings.TrimSpace(strings.TrimSpace(a.FirstName) + " " + strings.TrimSpace(a.LastName))
	}
	if name == "" || strings.TrimSpace(a.Province) == "" || strings.TrimSpace(a.City) == "" || strings.TrimSpace(a.District) == "" {
		return InquiryArtistContext{}, domain.ErrProfileIncomplete
	}
	return InquiryArtistContext{
		DisplayName: name,
		Province:    strings.TrimSpace(a.Province),
		City:        strings.TrimSpace(a.City),
		District:    strings.TrimSpace(a.District),
	}, nil
}

func referenceCode(now time.Time, id uuid.UUID) string {
	hex := strings.ToUpper(strings.ReplaceAll(id.String(), "-", ""))
	if len(hex) < 4 {
		hex = hex + "0000"
	}
	return fmt.Sprintf("MM%s-%s", now.UTC().Format("20060102"), hex[:4])
}

func toOrderCard(row store.Order, atts []store.OrderAttachment, role string) OrderCard {
	name, city := row.MakerName, row.MakerCity
	if role == "vendor" {
		name, city = row.ArtistDisplayName, row.ShipToCity
	}
	return OrderCard{
		ID:               row.ID.String(),
		ReferenceCode:    row.ReferenceCode,
		Status:           string(row.Status),
		ProjectName:      row.ProjectName,
		ProductName:      row.ProductName,
		CategorySlug:     row.CategorySlug,
		WantSample:       row.WantSample,
		Quantity:         row.Quantity,
		RequestedShipOn:  row.RequestedShipOn.UTC().Format("2006-01-02"),
		CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
		CounterpartyName: name,
		CounterpartyCity: city,
		ViewerRole:       role,
		PrimaryImage:     firstImage(atts),
	}
}

func toOrderDetail(row store.Order, atts []store.OrderAttachment, role string) OrderDetail {
	pub := make([]OrderAttachmentPublic, 0, len(atts))
	for _, a := range atts {
		pub = append(pub, OrderAttachmentPublic{
			URL:              a.Url,
			ContentType:      a.ContentType,
			OriginalFilename: a.OriginalFilename,
			ByteSize:         a.ByteSize,
		})
	}
	return OrderDetail{
		OrderCard:        toOrderCard(row, atts, role),
		MakerID:          row.MakerID.String(),
		QuantityNote:     row.QuantityNote,
		DesignCount:      row.DesignCount,
		SampleQuantity:   row.SampleQuantity,
		RequestedReadyOn: row.RequestedReadyOn.UTC().Format("2006-01-02"),
		Rush:             row.Rush,
		BudgetMinIDR:     row.BudgetMinIdr,
		BudgetMaxIDR:     row.BudgetMaxIdr,
		Notes:            row.Notes,
		ShipToProvince:   row.ShipToProvince,
		ShipToCity:       row.ShipToCity,
		ShipToDistrict:   row.ShipToDistrict,
		ShipToAddress:    row.ShipToAddress,
		VendorNote:       row.VendorNote,
		Attachments:      pub,
	}
}

func firstImage(atts []store.OrderAttachment) *Media {
	for _, a := range atts {
		if strings.HasPrefix(a.ContentType, "image/") {
			return &Media{URL: a.Url, Alt: a.OriginalFilename}
		}
	}
	return nil
}
