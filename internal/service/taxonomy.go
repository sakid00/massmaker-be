package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/store"
)

type CategoryProposal struct {
	Slug              string `json:"slug"`
	Kind              string `json:"kind"`
	LabelID           string `json:"labelId"`
	GroupSlug         string `json:"groupSlug,omitempty"`
	PublicationStatus string `json:"publicationStatus"`
}

type CategoryProposalList struct {
	Groups     []CategoryProposal `json:"groups"`
	Categories []CategoryProposal `json:"categories"`
}

func (s *MakerSelf) Propose(ctx context.Context, authorization, userID string, kind, label, groupSlug string) (*CategoryProposal, error) {
	profile, err := s.requireVendor(ctx, authorization)
	if err != nil {
		return nil, err
	}
	maker, err := s.ensureMaker(ctx, profile, userID)
	if err != nil {
		return nil, err
	}
	label = strings.TrimSpace(label)
	slug, err := domain.ProposeSlug(label)
	if err != nil {
		return nil, err
	}
	kind = strings.TrimSpace(kind)
	switch kind {
	case "group":
		return s.proposeGroup(ctx, maker.ID, slug, label)
	case "leaf":
		return s.proposeLeaf(ctx, maker.ID, slug, label, strings.TrimSpace(groupSlug))
	default:
		return nil, domain.NewAppError(400, "validation", "kind must be group or leaf")
	}
}

func (s *MakerSelf) ListProposals(ctx context.Context, authorization string) (*CategoryProposalList, error) {
	profile, err := s.requireVendor(ctx, authorization)
	if err != nil {
		return nil, err
	}
	vid, err := uuid.Parse(profile.Vendor.ID)
	if err != nil {
		return &CategoryProposalList{Groups: []CategoryProposal{}, Categories: []CategoryProposal{}}, nil
	}
	maker, err := s.db.Queries.GetMakerByEnmasseVendor(ctx, &vid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &CategoryProposalList{Groups: []CategoryProposal{}, Categories: []CategoryProposal{}}, nil
		}
		return nil, fmt.Errorf("proposals: %w", err)
	}
	groups, err := s.db.Queries.ListMakerDraftCategoryGroups(ctx, &maker.ID)
	if err != nil {
		return nil, fmt.Errorf("proposals groups: %w", err)
	}
	leaves, err := s.db.Queries.ListMakerDraftCategories(ctx, &maker.ID)
	if err != nil {
		return nil, fmt.Errorf("proposals leaves: %w", err)
	}
	out := &CategoryProposalList{
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

func (s *MakerSelf) proposeGroup(ctx context.Context, makerID uuid.UUID, slug, label string) (*CategoryProposal, error) {
	if _, err := s.db.Queries.GetCategoryGroup(ctx, slug); err == nil {
		return nil, domain.NewAppError(409, "conflict", "category already exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("propose group: %w", err)
	}
	if _, err := s.db.Queries.GetCategory(ctx, slug); err == nil {
		return nil, domain.NewAppError(409, "conflict", "category already exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("propose group: %w", err)
	}
	if _, err := s.db.Queries.GetCategoryGroupByLabel(ctx, label); err == nil {
		return nil, domain.NewAppError(409, "conflict", "category already exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("propose group label: %w", err)
	}
	row, err := s.db.Queries.InsertCategoryGroup(ctx, store.InsertCategoryGroupParams{
		Slug:              slug,
		LabelID:           label,
		ShortcutOrder:     100,
		PublicationStatus: store.PublicationStatusDraft,
		Source:            "vendor_proposal",
		ProposedByMakerID: &makerID,
	})
	if err != nil {
		return nil, mapTaxonomyWrite(err)
	}
	return &CategoryProposal{
		Slug:              row.Slug,
		Kind:              "group",
		LabelID:           row.LabelID,
		PublicationStatus: string(row.PublicationStatus),
	}, nil
}

func (s *MakerSelf) proposeLeaf(ctx context.Context, makerID uuid.UUID, slug, label, groupSlug string) (*CategoryProposal, error) {
	if groupSlug == "" {
		return nil, domain.NewAppError(400, "validation", "groupSlug is required")
	}
	group, err := s.db.Queries.GetCategoryGroup(ctx, groupSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewAppError(400, "validation", "unknown category group")
		}
		return nil, fmt.Errorf("propose leaf group: %w", err)
	}
	var proposed *string
	if group.ProposedByMakerID != nil {
		id := group.ProposedByMakerID.String()
		proposed = &id
	}
	if !domain.CapabilityAllowed(string(group.PublicationStatus), proposed, makerID.String()) {
		return nil, domain.NewAppError(400, "validation", "unknown category group")
	}
	if _, err := s.db.Queries.GetCategory(ctx, slug); err == nil {
		return nil, domain.NewAppError(409, "conflict", "category already exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("propose leaf: %w", err)
	}
	if _, err := s.db.Queries.GetCategoryGroup(ctx, slug); err == nil {
		return nil, domain.NewAppError(409, "conflict", "category already exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("propose leaf: %w", err)
	}
	if _, err := s.db.Queries.GetCategoryByLabel(ctx, label); err == nil {
		return nil, domain.NewAppError(409, "conflict", "category already exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("propose leaf label: %w", err)
	}
	row, err := s.db.Queries.InsertCategoryRow(ctx, store.InsertCategoryRowParams{
		Slug:              slug,
		LabelID:           label,
		ShortcutOrder:     100,
		GroupSlug:         groupSlug,
		PublicationStatus: store.PublicationStatusDraft,
		Source:            "vendor_proposal",
		ProposedByMakerID: &makerID,
	})
	if err != nil {
		return nil, mapTaxonomyWrite(err)
	}
	return &CategoryProposal{
		Slug:              row.Slug,
		Kind:              "leaf",
		LabelID:           row.LabelID,
		GroupSlug:         row.GroupSlug,
		PublicationStatus: string(row.PublicationStatus),
	}, nil
}

func mapTaxonomyWrite(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return domain.NewAppError(409, "conflict", "category already exists")
	}
	return fmt.Errorf("taxonomy write: %w", err)
}
