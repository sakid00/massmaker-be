package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sakid00/massmaker-be/internal/auth"
	"github.com/sakid00/massmaker-be/internal/config"
	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/enmasse"
	"github.com/sakid00/massmaker-be/internal/store"
)

type Admin struct {
	db  *store.DB
	cfg *config.Config
	em  *enmasse.Client
}

func NewAdmin(db *store.DB, cfg *config.Config, em *enmasse.Client) *Admin {
	return &Admin{db: db, cfg: cfg, em: em}
}

func (s *Admin) Bootstrap(ctx context.Context) error {
	n, err := s.db.Queries.CountStaff(ctx)
	if err != nil {
		return fmt.Errorf("staff count: %w", err)
	}
	if n > 0 {
		return nil
	}
	_, err = s.db.Queries.InsertStaff(ctx, store.InsertStaffParams{
		Email:        strings.ToLower(strings.TrimSpace(s.cfg.StaffEmail)),
		PasswordHash: s.cfg.StaffPassword,
		Role:         "content_owner",
	})
	return err
}

func (s *Admin) Login(email, password string) (string, error) {
	row, err := s.db.Queries.GetStaffByEmail(context.Background(), strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrUnauthorized
		}
		return "", fmt.Errorf("staff login: %w", err)
	}
	if !auth.Verify(row.PasswordHash, password) {
		return "", domain.ErrUnauthorized
	}
	return auth.MintStaff(s.cfg.StaffJWTSecret, row.Email, s.cfg.StaffJWTExpiry)
}

func (s *Admin) ParseStaff(token string) error {
	return auth.ParseStaff(s.cfg.StaffJWTSecret, token)
}

type CreateMakerInput struct {
	Name             string
	City             string
	VerificationDate string
	Capabilities     []string
	MOQQuantity      *int32
	MOQBasis         string
	MOQBasisNote     *string
	LeadTimeEstimate *string
	PriceMinIDR      *int32
	PriceMaxIDR      *int32
	PriceNote        *string
	Conditions       *string
	EnmasseVendorID  *uuid.UUID
}

func (s *Admin) CreateMaker(ctx context.Context, in CreateMakerInput) (uuid.UUID, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.City) == "" {
		return uuid.Nil, domain.NewAppError(400, "validation", "name and city are required")
	}
	var vdate *time.Time
	if in.VerificationDate != "" {
		t, err := time.Parse("2006-01-02", in.VerificationDate)
		if err != nil {
			return uuid.Nil, domain.NewAppError(400, "validation", "verificationDate must be YYYY-MM-DD")
		}
		vdate = &t
	}
	var basis store.NullMoqBasis
	if in.MOQBasis != "" {
		basis = store.NullMoqBasis{MoqBasis: store.MoqBasis(in.MOQBasis), Valid: true}
	}
	row, err := s.db.Queries.InsertMaker(ctx, store.InsertMakerParams{
		Name:             strings.TrimSpace(in.Name),
		City:             strings.TrimSpace(in.City),
		VerificationDate: vdate,
		MoqQuantity:      in.MOQQuantity,
		MoqBasis:         basis,
		MoqBasisNote:     in.MOQBasisNote,
		LeadTimeEstimate: in.LeadTimeEstimate,
		PriceMinIdr:      in.PriceMinIDR,
		PriceMaxIdr:      in.PriceMaxIDR,
		PriceNote:        in.PriceNote,
		Conditions:       in.Conditions,
		EnmasseVendorID:  in.EnmasseVendorID,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create maker: %w", err)
	}
	if err := s.replaceCaps(ctx, row.ID, in.Capabilities); err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}

func (s *Admin) Publish(ctx context.Context, id uuid.UUID) error {
	row, err := s.db.Queries.GetMaker(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.City) == "" || row.VerificationDate == nil {
		return domain.NewAppError(400, "validation", "name, city, and verificationDate are required to publish")
	}
	caps, err := s.db.Queries.ListMakerCapabilities(ctx, []uuid.UUID{id})
	if err != nil {
		return err
	}
	publishedCaps := 0
	for _, cap := range caps {
		if cap.PublicationStatus == store.PublicationStatusPublished {
			publishedCaps++
		}
	}
	if publishedCaps == 0 {
		return domain.NewAppError(400, "validation", "at least one published capability is required to publish")
	}
	if row.EnmasseVendorID != nil {
		if !OverlayReady(row.PriceMinIdr, row.PriceMaxIdr, deref(row.HoursOpen), deref(row.HoursClose), row.HoursDays, row.MoqQuantity, moqBasis(row.MoqBasis), deref(row.LeadTimeEstimate), len(caps)) {
			return domain.NewAppError(400, "validation", "vendor overlay is not ready for review")
		}
		if s.em != nil {
			vendor, err := s.em.InternalVendor(ctx, row.EnmasseVendorID.String())
			if err != nil {
				return err
			}
			if !vendorIdentityReady(vendor) {
				return domain.NewAppError(400, "validation", "vendor identity is incomplete")
			}
		}
	}
	return s.db.Queries.SetPublication(ctx, store.SetPublicationParams{
		ID:                id,
		PublicationStatus: store.PublicationStatusPublished,
	})
}

func (s *Admin) Withdraw(ctx context.Context, id uuid.UUID) error {
	if _, err := s.db.Queries.GetMaker(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return s.db.Queries.SetPublication(ctx, store.SetPublicationParams{
		ID:                id,
		PublicationStatus: store.PublicationStatusWithdrawn,
	})
}

func (s *Admin) PutContact(ctx context.Context, makerID uuid.UUID, channel, digits string) error {
	if strings.TrimSpace(digits) == "" {
		return domain.NewAppError(400, "validation", "e164Digits required")
	}
	if channel == "" {
		channel = "whatsapp"
	}
	_, err := s.db.Queries.UpsertContact(ctx, store.UpsertContactParams{
		MakerID:    makerID,
		Channel:    channel,
		E164Digits: strings.TrimSpace(digits),
	})
	return err
}

func (s *Admin) ConsentContact(ctx context.Context, makerID uuid.UUID) error {
	return s.db.Queries.PermitContact(ctx, makerID)
}

func (s *Admin) ListMakers(ctx context.Context) ([]store.ListAdminMakersRow, error) {
	return s.db.Queries.ListAdminMakers(ctx)
}

func (s *Admin) replaceCaps(ctx context.Context, makerID uuid.UUID, slugs []string) error {
	return replaceCaps(ctx, s.db, makerID, slugs)
}

type TaxonomyDrafts struct {
	Groups     []CategoryProposal `json:"groups"`
	Categories []CategoryProposal `json:"categories"`
}

func (s *Admin) ListTaxonomyDrafts(ctx context.Context) (*TaxonomyDrafts, error) {
	groups, err := s.db.Queries.ListDraftCategoryGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("taxonomy drafts groups: %w", err)
	}
	leaves, err := s.db.Queries.ListDraftCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("taxonomy drafts leaves: %w", err)
	}
	out := &TaxonomyDrafts{
		Groups:     make([]CategoryProposal, 0, len(groups)),
		Categories: make([]CategoryProposal, 0, len(leaves)),
	}
	for _, g := range groups {
		out.Groups = append(out.Groups, CategoryProposal{
			Slug:              g.Slug,
			Kind:              "group",
			LabelID:           g.LabelID,
			PublicationStatus: string(g.PublicationStatus),
		})
	}
	for _, c := range leaves {
		out.Categories = append(out.Categories, CategoryProposal{
			Slug:              c.Slug,
			Kind:              "leaf",
			LabelID:           c.LabelID,
			GroupSlug:         c.GroupSlug,
			PublicationStatus: string(c.PublicationStatus),
		})
	}
	return out, nil
}

func (s *Admin) SetGroupPublication(ctx context.Context, slug, status string) error {
	if _, err := s.db.Queries.GetCategoryGroup(ctx, slug); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	pub, err := parseTaxonomyStatus(status)
	if err != nil {
		return err
	}
	return s.db.Queries.SetCategoryGroupPublication(ctx, store.SetCategoryGroupPublicationParams{
		Slug:              slug,
		PublicationStatus: pub,
	})
}

func (s *Admin) SetLeafPublication(ctx context.Context, slug, status string) error {
	row, err := s.db.Queries.GetCategory(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	pub, err := parseTaxonomyStatus(status)
	if err != nil {
		return err
	}
	if err := s.db.Queries.SetCategoryPublication(ctx, store.SetCategoryPublicationParams{
		Slug:              slug,
		PublicationStatus: pub,
	}); err != nil {
		return err
	}
	if pub == store.PublicationStatusPublished {
		if err := s.db.Queries.InsertSynonym(ctx, store.InsertSynonymParams{
			CategorySlug: slug,
			Synonym:      slug,
		}); err != nil {
			return fmt.Errorf("taxonomy synonym slug: %w", err)
		}
		labelSyn := domain.NormalizeQuery(row.LabelID)
		if labelSyn != "" && labelSyn != slug {
			if err := s.db.Queries.InsertSynonym(ctx, store.InsertSynonymParams{
				CategorySlug: slug,
				Synonym:      labelSyn,
			}); err != nil {
				return fmt.Errorf("taxonomy synonym label: %w", err)
			}
		}
	}
	return nil
}

func parseTaxonomyStatus(status string) (store.PublicationStatus, error) {
	switch status {
	case "published":
		return store.PublicationStatusPublished, nil
	case "withdrawn":
		return store.PublicationStatusWithdrawn, nil
	default:
		return "", domain.NewAppError(400, "validation", "status must be published or withdrawn")
	}
}

func vendorIdentityReady(v *enmasse.InternalVendor) bool {
	if v == nil {
		return false
	}
	return strings.TrimSpace(v.Name) != "" &&
		strings.TrimSpace(v.Province) != "" &&
		strings.TrimSpace(v.City) != "" &&
		strings.TrimSpace(v.District) != "" &&
		strings.TrimSpace(v.PICName) != "" &&
		strings.TrimSpace(v.Address) != "" &&
		strings.TrimSpace(v.WhatsApp) != "" &&
		strings.TrimSpace(v.Bio) != ""
}
