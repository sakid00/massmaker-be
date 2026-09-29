package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/enmasse"
	"github.com/sakid00/massmaker-be/internal/store"
)

const (
	missingCategory   = "category"
	missingPriceRange = "price_range"
	missingHours      = "hours"
	missingMOQ        = "moq"
	missingLeadTime   = "lead_time"
	maxPortfolioItems = 20
)

type MakerSelf struct {
	db *store.DB
	em *enmasse.Client
}

func NewMakerSelf(db *store.DB, em *enmasse.Client) *MakerSelf {
	return &MakerSelf{db: db, em: em}
}

type PortfolioItem struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type MakerOverlay struct {
	ID               string          `json:"id,omitempty"`
	Capabilities     []string        `json:"capabilities"`
	MOQ              *MOQ            `json:"moq"`
	LeadTimeEstimate *string         `json:"leadTimeEstimate"`
	PriceMinIDR      *int32          `json:"priceMinIdr"`
	PriceMaxIDR      *int32          `json:"priceMaxIdr"`
	PriceNote        *string         `json:"priceNote"`
	HoursOpen        string          `json:"hoursOpen"`
	HoursClose       string          `json:"hoursClose"`
	HoursDays        []string        `json:"hoursDays"`
	Portfolio        []PortfolioItem `json:"portfolio"`
	ReadyForReview   bool            `json:"readyForReview"`
	Missing          []string        `json:"missing"`
}

type Onboarding struct {
	Role            string          `json:"role"`
	ProfileComplete bool            `json:"profileComplete"`
	Missing         []string        `json:"missing"`
	ReadyForReview  bool            `json:"readyForReview"`
	Profile         json.RawMessage `json:"profile"`
	Maker           *MakerOverlay   `json:"maker,omitempty"`
}

type PatchMakerInput struct {
	Capabilities     []string
	MOQ              *MOQ
	LeadTimeEstimate *string
	PriceMinIDR      *int32
	PriceMaxIDR      *int32
	PriceNote        *string
	HoursOpen        *string
	HoursClose       *string
	HoursDays        []string
	HoursDaysSet     bool
	Portfolio        *[]PortfolioItem
}

func (s *MakerSelf) Onboarding(ctx context.Context, authorization string) (*Onboarding, error) {
	if s.em == nil {
		return nil, domain.ErrUnavailable
	}
	profile, err := s.em.MeProfile(ctx, authorization)
	if err != nil {
		return nil, err
	}
	raw := profile.Raw
	if len(raw) == 0 {
		encoded, err := json.Marshal(profile)
		if err != nil {
			return nil, fmt.Errorf("onboarding: profile: %w", err)
		}
		raw = encoded
	}
	out := &Onboarding{
		Role:            profile.Role,
		ProfileComplete: profile.ProfileComplete,
		Missing:         append([]string{}, profile.Missing...),
		ReadyForReview:  false,
		Profile:         raw,
	}
	if profile.Role != "vendor" {
		return out, nil
	}
	overlay, err := s.overlayForVendor(ctx, profile)
	if err != nil {
		return nil, err
	}
	out.Maker = overlay
	out.Missing = append(out.Missing, overlay.Missing...)
	out.ReadyForReview = profile.ProfileComplete && overlay.ReadyForReview
	out.ProfileComplete = out.ReadyForReview
	return out, nil
}

func (s *MakerSelf) Get(ctx context.Context, authorization string) (*MakerOverlay, error) {
	profile, err := s.requireVendor(ctx, authorization)
	if err != nil {
		return nil, err
	}
	return s.overlayForVendor(ctx, profile)
}

func (s *MakerSelf) Patch(ctx context.Context, authorization, userID string, in PatchMakerInput) (*MakerOverlay, error) {
	profile, err := s.requireVendor(ctx, authorization)
	if err != nil {
		return nil, err
	}
	maker, err := s.ensureMaker(ctx, profile, userID)
	if err != nil {
		return nil, err
	}

	hoursOpen := pickPtr(in.HoursOpen, maker.HoursOpen)
	hoursClose := pickPtr(in.HoursClose, maker.HoursClose)
	hoursDays := maker.HoursDays
	if hoursDays == nil {
		hoursDays = []string{}
	}
	if in.HoursDaysSet {
		days, err := domain.NormalizeHoursDays(in.HoursDays)
		if err != nil {
			return nil, err
		}
		hoursDays = days
	}
	min := maker.PriceMinIdr
	max := maker.PriceMaxIdr
	note := maker.PriceNote
	if in.PriceMinIDR != nil {
		min = in.PriceMinIDR
	}
	if in.PriceMaxIDR != nil {
		max = in.PriceMaxIDR
	}
	if in.PriceNote != nil {
		note = strPtr(strings.TrimSpace(*in.PriceNote))
	}
	moqQty := maker.MoqQuantity
	moqBasisVal := maker.MoqBasis
	moqNote := maker.MoqBasisNote
	if in.MOQ != nil {
		qty := in.MOQ.Quantity
		moqQty = &qty
		basis, err := parseMoqBasis(in.MOQ.Basis)
		if err != nil {
			return nil, err
		}
		moqBasisVal = basis
		if in.MOQ.BasisNote != nil {
			moqNote = strPtr(strings.TrimSpace(*in.MOQ.BasisNote))
		} else {
			moqNote = nil
		}
	}
	lead := maker.LeadTimeEstimate
	if in.LeadTimeEstimate != nil {
		lead = strPtr(strings.TrimSpace(*in.LeadTimeEstimate))
	}

	if err := s.db.Queries.UpdateMakerOverlay(ctx, store.UpdateMakerOverlayParams{
		ID:               maker.ID,
		HoursOpen:        hoursOpen,
		HoursClose:       hoursClose,
		HoursDays:        hoursDays,
		PriceMinIdr:      min,
		PriceMaxIdr:      max,
		PriceNote:        note,
		MoqQuantity:      moqQty,
		MoqBasis:         moqBasisVal,
		MoqBasisNote:     moqNote,
		LeadTimeEstimate: lead,
	}); err != nil {
		return nil, fmt.Errorf("maker overlay: update: %w", err)
	}
	if err := s.db.Queries.UpdateMakerIdentity(ctx, store.UpdateMakerIdentityParams{
		ID:   maker.ID,
		Name: strings.TrimSpace(profile.Vendor.Name),
		City: strings.TrimSpace(profile.Vendor.City),
	}); err != nil {
		return nil, fmt.Errorf("maker overlay: identity: %w", err)
	}

	if in.Capabilities != nil {
		if err := replaceCaps(ctx, s.db, maker.ID, in.Capabilities); err != nil {
			return nil, err
		}
	}
	if in.Portfolio != nil {
		if len(*in.Portfolio) > maxPortfolioItems {
			return nil, domain.NewAppError(400, "validation", "at most 20 portfolio items")
		}
		if err := s.db.Queries.DeleteMakerMedia(ctx, maker.ID); err != nil {
			return nil, fmt.Errorf("maker overlay: media: %w", err)
		}
		for i, item := range *in.Portfolio {
			url := strings.TrimSpace(item.URL)
			if url == "" {
				continue
			}
			if err := s.db.Queries.InsertMakerMedia(ctx, store.InsertMakerMediaParams{
				MakerID:   maker.ID,
				Url:       url,
				Alt:       strings.TrimSpace(item.Alt),
				SortOrder: int32(i),
			}); err != nil {
				return nil, fmt.Errorf("maker overlay: media insert: %w", err)
			}
		}
	}

	if wa := strings.TrimSpace(profile.Vendor.WhatsApp); wa != "" {
		digits := strings.TrimPrefix(wa, "+")
		if _, err := s.db.Queries.UpsertContact(ctx, store.UpsertContactParams{
			MakerID:    maker.ID,
			Channel:    "whatsapp",
			E164Digits: digits,
		}); err != nil {
			return nil, fmt.Errorf("maker overlay: contact: %w", err)
		}
	}

	return s.overlayForVendor(ctx, profile)
}

func (s *MakerSelf) ensureMaker(ctx context.Context, profile *enmasse.MeProfile, userID string) (store.GetMakerByEnmasseVendorRow, error) {
	vid, err := uuid.Parse(profile.Vendor.ID)
	if err != nil {
		return store.GetMakerByEnmasseVendorRow{}, domain.ErrBadRequest
	}
	maker, err := s.db.Queries.GetMakerByEnmasseVendor(ctx, &vid)
	if err == nil {
		if err := s.bindMakerUser(ctx, maker.ID, userID); err != nil {
			return store.GetMakerByEnmasseVendorRow{}, err
		}
		return maker, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return store.GetMakerByEnmasseVendorRow{}, fmt.Errorf("maker overlay: get: %w", err)
	}
	created, err := s.db.Queries.InsertMaker(ctx, store.InsertMakerParams{
		Name:            strings.TrimSpace(profile.Vendor.Name),
		City:            strings.TrimSpace(profile.Vendor.City),
		EnmasseVendorID: &vid,
	})
	if err != nil {
		return store.GetMakerByEnmasseVendorRow{}, fmt.Errorf("maker overlay: create: %w", err)
	}
	if err := s.bindMakerUser(ctx, created.ID, userID); err != nil {
		return store.GetMakerByEnmasseVendorRow{}, err
	}
	return store.GetMakerByEnmasseVendorRow{
		ID:               created.ID,
		Name:             created.Name,
		City:             created.City,
		MoqQuantity:      created.MoqQuantity,
		MoqBasis:         created.MoqBasis,
		MoqBasisNote:     created.MoqBasisNote,
		LeadTimeEstimate: created.LeadTimeEstimate,
		PriceMinIdr:      created.PriceMinIdr,
		PriceMaxIdr:      created.PriceMaxIdr,
		PriceNote:        created.PriceNote,
		HoursOpen:        created.HoursOpen,
		HoursClose:       created.HoursClose,
		HoursDays:        created.HoursDays,
		ServiceArea:      created.ServiceArea,
		EnmasseVendorID:  created.EnmasseVendorID,
		EnmasseUserID:    created.EnmasseUserID,
	}, nil
}

func (s *MakerSelf) bindMakerUser(ctx context.Context, makerID uuid.UUID, userID string) error {
	uid, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return nil
	}
	if err := s.db.Queries.SetMakerEnmasseUserID(ctx, store.SetMakerEnmasseUserIDParams{
		ID:            makerID,
		EnmasseUserID: &uid,
	}); err != nil {
		return fmt.Errorf("maker overlay: bind user: %w", err)
	}
	return nil
}

func (s *MakerSelf) requireVendor(ctx context.Context, authorization string) (*enmasse.MeProfile, error) {
	if s.em == nil {
		return nil, domain.ErrUnavailable
	}
	profile, err := s.em.MeProfile(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if profile.Role != "vendor" || profile.Vendor == nil {
		return nil, domain.ErrForbidden
	}
	return profile, nil
}

func (s *MakerSelf) overlayForVendor(ctx context.Context, profile *enmasse.MeProfile) (*MakerOverlay, error) {
	out := &MakerOverlay{Capabilities: []string{}, HoursDays: []string{}, Portfolio: []PortfolioItem{}, Missing: []string{}}
	vid, err := uuid.Parse(profile.Vendor.ID)
	if err != nil {
		return out, nil
	}
	maker, err := s.db.Queries.GetMakerByEnmasseVendor(ctx, &vid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			out.Missing = append([]string{missingCategory}, overlayMissing(nil, nil, "", "", nil, nil, "", "")...)
			return out, nil
		}
		return nil, fmt.Errorf("maker overlay: %w", err)
	}
	out.ID = maker.ID.String()
	out.MOQ = toMOQ(maker.MoqQuantity, moqBasis(maker.MoqBasis), maker.MoqBasisNote)
	out.LeadTimeEstimate = maker.LeadTimeEstimate
	out.PriceMinIDR = maker.PriceMinIdr
	out.PriceMaxIDR = maker.PriceMaxIdr
	out.PriceNote = maker.PriceNote
	out.HoursOpen = deref(maker.HoursOpen)
	out.HoursClose = deref(maker.HoursClose)
	out.HoursDays = maker.HoursDays
	if out.HoursDays == nil {
		out.HoursDays = []string{}
	}

	caps, err := s.db.Queries.ListMakerCapabilities(ctx, []uuid.UUID{maker.ID})
	if err != nil {
		return nil, err
	}
	for _, c := range caps {
		out.Capabilities = append(out.Capabilities, c.CategorySlug)
	}
	media, err := s.db.Queries.ListMakerMedia(ctx, maker.ID)
	if err != nil {
		return nil, err
	}
	for _, m := range media {
		out.Portfolio = append(out.Portfolio, PortfolioItem{URL: m.Url, Alt: m.Alt})
	}
	out.Missing = overlayMissing(
		maker.PriceMinIdr,
		maker.PriceMaxIdr,
		out.HoursOpen,
		out.HoursClose,
		out.HoursDays,
		maker.MoqQuantity,
		moqBasis(maker.MoqBasis),
		deref(maker.LeadTimeEstimate),
	)
	if len(out.Capabilities) == 0 {
		out.Missing = append([]string{missingCategory}, out.Missing...)
	}
	out.ReadyForReview = len(out.Missing) == 0
	if out.Missing == nil {
		out.Missing = []string{}
	}
	return out, nil
}

func OverlayReady(min, max *int32, hoursOpen, hoursClose string, hoursDays []string, moqQty *int32, moqBasis string, leadTime string, caps int) bool {
	return len(overlayMissing(min, max, hoursOpen, hoursClose, hoursDays, moqQty, moqBasis, leadTime)) == 0 && caps > 0
}

func overlayMissing(min, max *int32, hoursOpen, hoursClose string, hoursDays []string, moqQty *int32, moqBasis string, leadTime string) []string {
	var missing []string
	if min == nil || max == nil {
		missing = append(missing, missingPriceRange)
	}
	if strings.TrimSpace(hoursOpen) == "" || strings.TrimSpace(hoursClose) == "" || len(hoursDays) == 0 {
		missing = append(missing, missingHours)
	}
	if moqQty == nil || *moqQty <= 0 || !knownMoqBasis(moqBasis) {
		missing = append(missing, missingMOQ)
	}
	if strings.TrimSpace(leadTime) == "" {
		missing = append(missing, missingLeadTime)
	}
	return missing
}

func parseMoqBasis(basis string) (store.NullMoqBasis, error) {
	basis = strings.TrimSpace(basis)
	if basis == "" {
		return store.NullMoqBasis{}, nil
	}
	if !knownMoqBasis(basis) {
		return store.NullMoqBasis{}, domain.NewAppError(400, "validation", "unknown moq basis")
	}
	return store.NullMoqBasis{MoqBasis: store.MoqBasis(basis), Valid: true}, nil
}

func knownMoqBasis(basis string) bool {
	switch store.MoqBasis(strings.TrimSpace(basis)) {
	case store.MoqBasisTotal, store.MoqBasisPerDesain, store.MoqBasisPerWarna, store.MoqBasisOther:
		return true
	default:
		return false
	}
}

func replaceCaps(ctx context.Context, db *store.DB, makerID uuid.UUID, slugs []string) error {
	known, err := db.Queries.ListAllCategories(ctx)
	if err != nil {
		return err
	}
	allowed := make(map[string]struct{}, len(known))
	maker := makerID.String()
	for _, c := range known {
		var proposed *string
		if c.ProposedByMakerID != nil {
			id := c.ProposedByMakerID.String()
			proposed = &id
		}
		if domain.CapabilityAllowed(string(c.PublicationStatus), proposed, maker) {
			allowed[c.Slug] = struct{}{}
		}
	}
	if err := db.Queries.ReplaceCapabilities(ctx, makerID); err != nil {
		return err
	}
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if _, ok := allowed[slug]; !ok {
			return domain.NewAppError(400, "validation", "unknown category")
		}
		if err := db.Queries.InsertCapability(ctx, store.InsertCapabilityParams{
			MakerID:      makerID,
			CategorySlug: slug,
		}); err != nil {
			return fmt.Errorf("capability %s: %w", slug, err)
		}
	}
	return nil
}

func pickPtr(override *string, current *string) *string {
	if override == nil {
		return current
	}
	return strPtr(strings.TrimSpace(*override))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
