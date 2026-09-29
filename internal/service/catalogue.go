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

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/enmasse"
	"github.com/sakid00/massmaker-be/internal/store"
)

type Catalogue struct {
	db      *store.DB
	enmasse *enmasse.Client
}

func NewCatalogue(db *store.DB, em *enmasse.Client) *Catalogue {
	return &Catalogue{db: db, enmasse: em}
}

type CategoryPublic struct {
	Slug          string `json:"slug"`
	LabelID       string `json:"labelId"`
	ShortcutOrder int32  `json:"shortcutOrder"`
	GroupSlug     string `json:"groupSlug"`
}

type CategoryGroupPublic struct {
	Slug          string `json:"slug"`
	LabelID       string `json:"labelId"`
	ShortcutOrder int32  `json:"shortcutOrder"`
}

type CategoryList struct {
	CategoryGroups []CategoryGroupPublic `json:"categoryGroups"`
	Categories     []CategoryPublic      `json:"categories"`
	CatalogueReady bool                  `json:"catalogueReady"`
}

type MakerCard struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	City                string  `json:"city"`
	RelevantCapability  string  `json:"relevantCapability"`
	MOQ                 *MOQ    `json:"moq"`
	LeadTimeEstimate    *string `json:"leadTimeEstimate"`
	PriceRange          *Price  `json:"priceRange"`
	VerificationDate    string  `json:"verificationDate"`
	NeedsReconfirmation bool    `json:"needsReconfirmation"`
	PrimaryImage        *Media  `json:"primaryImage"`
}

type MOQ struct {
	Quantity  int32   `json:"quantity"`
	Basis     string  `json:"basis"`
	BasisNote *string `json:"basisNote"`
}

type Price struct {
	MinIDR int32   `json:"minIdr"`
	MaxIDR int32   `json:"maxIdr"`
	Note   *string `json:"note"`
}

type Media struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type SearchResponse struct {
	Interpretation      SearchInterpretation `json:"interpretation"`
	Makers              []MakerCard          `json:"makers"`
	AvailableCategories []CategoryPublic     `json:"availableCategories"`
}

type SearchInterpretation struct {
	Query          string  `json:"query"`
	Category       *string `json:"category"`
	MatchedSynonym *string `json:"matchedSynonym"`
	NoResultReason *string `json:"noResultReason"`
}

type MakerDetail struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	City                string           `json:"city"`
	Capabilities        []Capability     `json:"capabilities"`
	MOQ                 *MOQ             `json:"moq"`
	LeadTimeEstimate    *string          `json:"leadTimeEstimate"`
	PriceRange          *Price           `json:"priceRange"`
	Conditions          *string          `json:"conditions"`
	VerificationDate    string           `json:"verificationDate"`
	NeedsReconfirmation bool             `json:"needsReconfirmation"`
	Images              []Media          `json:"images"`
	Recommendations     []Recommendation `json:"recommendations"`
	Badges              []any            `json:"badges"`
	Contact             *Contact         `json:"contact"`
	RecommendationEmpty bool             `json:"recommendationEmpty"`
}

type Capability struct {
	CategorySlug string `json:"categorySlug"`
	LabelID      string `json:"labelId"`
}

type Recommendation struct {
	ID                   string  `json:"id"`
	ArtistDisplayName    string  `json:"artistDisplayName"`
	JicafYear            int     `json:"jicafYear"`
	ProductMade          string  `json:"productMade"`
	ProductionDateApprox string  `json:"productionDateApprox"`
	Text                 string  `json:"text"`
	Quantity             *string `json:"quantity"`
	WouldUseAgain        *bool   `json:"wouldUseAgain"`
	Photos               []Media `json:"photos"`
	SourceLineID         string  `json:"sourceLineId"`
}

type Contact struct {
	Channel string `json:"channel"`
	WaMeURL string `json:"waMeUrl"`
}

type MakerUnavailable struct {
	ID         *string `json:"id"`
	Status     string  `json:"status"`
	MessageID  string  `json:"messageId"`
	Contact    any     `json:"contact"`
	SearchHref string  `json:"searchHref"`
}

func (s *Catalogue) Categories(ctx context.Context) (*CategoryList, error) {
	rows, err := s.db.Queries.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("categories: %w", err)
	}
	groups, err := s.db.Queries.ListCategoryGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("category groups: %w", err)
	}
	n, err := s.db.Queries.CountPublishedMakers(ctx)
	if err != nil {
		return nil, fmt.Errorf("categories count: %w", err)
	}
	out := &CategoryList{
		CategoryGroups: toCategoryGroups(groups),
		Categories:     toCategories(rows),
		CatalogueReady: n > 0,
	}
	return out, nil
}

func (s *Catalogue) Search(ctx context.Context, q, category string) (*SearchResponse, error) {
	cats, err := s.Categories(ctx)
	if err != nil {
		return nil, err
	}
	q = domain.NormalizeQuery(q)
	category = strings.TrimSpace(category)

	interp := SearchInterpretation{Query: q}
	if category == "" && q != "" {
		syn, err := s.db.Queries.GetSynonym(ctx, q)
		if err == nil {
			category = syn.CategorySlug
			interp.MatchedSynonym = &syn.Synonym
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("search synonym: %w", err)
		}
	}
	if category != "" && !publishedCategory(cats.Categories, category) {
		reason := "unrecognized"
		interp.NoResultReason = &reason
		return &SearchResponse{Interpretation: interp, Makers: []MakerCard{}, AvailableCategories: cats.Categories}, nil
	}
	if category != "" {
		interp.Category = &category
	}

	if !cats.CatalogueReady {
		reason := "empty_catalogue"
		interp.NoResultReason = &reason
		return &SearchResponse{Interpretation: interp, Makers: []MakerCard{}, AvailableCategories: cats.Categories}, nil
	}

	if category == "" && q != "" {
		reason := "unrecognized"
		interp.NoResultReason = &reason
		return &SearchResponse{Interpretation: interp, Makers: []MakerCard{}, AvailableCategories: cats.Categories}, nil
	}

	var catArg *string
	if category != "" {
		catArg = &category
	}
	rows, err := s.db.Queries.ListPublishedMakers(ctx, catArg)
	if err != nil {
		return nil, fmt.Errorf("search makers: %w", err)
	}
	cards, err := s.cards(ctx, rows, category)
	if err != nil {
		return nil, err
	}
	if len(cards) == 0 && category != "" {
		reason := "category_empty"
		interp.NoResultReason = &reason
	}
	return &SearchResponse{Interpretation: interp, Makers: cards, AvailableCategories: cats.Categories}, nil
}

func (s *Catalogue) Maker(ctx context.Context, id uuid.UUID) (*MakerDetail, *MakerUnavailable, int, error) {
	row, err := s.db.Queries.GetMaker(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unavailable(nil, "not_found"), http.StatusNotFound, nil
		}
		return nil, nil, 0, fmt.Errorf("maker: %w", err)
	}
	code := domain.PublicMakerCode(string(row.PublicationStatus), row.IsDemoFixture)
	sid := id.String()
	if code == http.StatusGone {
		return nil, unavailable(&sid, "withdrawn"), code, nil
	}
	if code != http.StatusOK {
		return nil, unavailable(&sid, "not_found"), code, nil
	}

	caps, err := s.db.Queries.ListMakerCapabilities(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("maker caps: %w", err)
	}
	recs, err := s.db.Queries.ListPublishedRecommendations(ctx, id)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("maker recs: %w", err)
	}

	vdate := dateString(row.VerificationDate)
	now := time.Now()
	detail := &MakerDetail{
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
		Recommendations:     toRecs(recs),
		Badges:              []any{},
		Contact:             nil,
	}
	detail.RecommendationEmpty = len(detail.Recommendations) == 0
	return detail, nil, http.StatusOK, nil
}

func (s *Catalogue) MeVendors(ctx context.Context, userID, role string) ([]MakerCard, error) {
	if role != "artist" {
		return nil, domain.ErrForbidden
	}
	st, err := s.enmasse.ProfileStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !st.Complete {
		return nil, domain.ErrProfileIncomplete
	}
	rows, err := s.db.Queries.ListPublishedMakers(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("me vendors: %w", err)
	}
	return s.cards(ctx, rows, "")
}

func (s *Catalogue) cards(ctx context.Context, rows []store.ListPublishedMakersRow, category string) ([]MakerCard, error) {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	capMap := map[uuid.UUID][]store.ListMakerCapabilitiesRow{}
	if len(ids) > 0 {
		caps, err := s.db.Queries.ListMakerCapabilities(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("cards caps: %w", err)
		}
		for _, c := range caps {
			capMap[c.MakerID] = append(capMap[c.MakerID], c)
		}
	}
	now := time.Now()
	out := make([]MakerCard, 0, len(rows))
	for _, r := range rows {
		label := ""
		for _, c := range capMap[r.ID] {
			if category == "" || c.CategorySlug == category {
				label = c.LabelID
				break
			}
		}
		if label == "" && len(capMap[r.ID]) > 0 {
			label = capMap[r.ID][0].LabelID
		}
		vdate := dateString(r.VerificationDate)
		out = append(out, MakerCard{
			ID:                  r.ID.String(),
			Name:                r.Name,
			City:                r.City,
			RelevantCapability:  label,
			MOQ:                 toMOQ(r.MoqQuantity, moqBasis(r.MoqBasis), r.MoqBasisNote),
			LeadTimeEstimate:    r.LeadTimeEstimate,
			PriceRange:          toPrice(r.PriceMinIdr, r.PriceMaxIdr, r.PriceNote),
			VerificationDate:    vdate,
			NeedsReconfirmation: domain.NeedsReconfirmation(vdate, now.Year(), int(now.Month()), now.Day()),
			PrimaryImage:        nil,
		})
	}
	return out, nil
}

func toCategories(rows []store.ListCategoriesRow) []CategoryPublic {
	out := make([]CategoryPublic, 0, len(rows))
	for _, r := range rows {
		out = append(out, CategoryPublic{
			Slug:          r.Slug,
			LabelID:       r.LabelID,
			ShortcutOrder: r.ShortcutOrder,
			GroupSlug:     r.GroupSlug,
		})
	}
	return out
}

func toCategoryGroups(rows []store.ListCategoryGroupsRow) []CategoryGroupPublic {
	out := make([]CategoryGroupPublic, 0, len(rows))
	for _, r := range rows {
		out = append(out, CategoryGroupPublic{Slug: r.Slug, LabelID: r.LabelID, ShortcutOrder: r.ShortcutOrder})
	}
	return out
}

func publishedCategory(rows []CategoryPublic, slug string) bool {
	for _, r := range rows {
		if r.Slug == slug {
			return true
		}
	}
	return false
}

func toCaps(rows []store.ListMakerCapabilitiesRow) []Capability {
	out := make([]Capability, 0, len(rows))
	for _, r := range rows {
		out = append(out, Capability{CategorySlug: r.CategorySlug, LabelID: r.LabelID})
	}
	return out
}

func toRecs(rows []store.ListPublishedRecommendationsRow) []Recommendation {
	out := make([]Recommendation, 0, len(rows))
	for _, r := range rows {
		out = append(out, Recommendation{
			ID:                   r.ID.String(),
			ArtistDisplayName:    r.DisplayName,
			JicafYear:            2026,
			ProductMade:          r.ProductMade,
			ProductionDateApprox: r.ProductionDateApprox,
			Text:                 r.Body,
			Quantity:             r.Quantity,
			WouldUseAgain:        r.WouldUseAgain,
			Photos:               []Media{},
			SourceLineID:         "interview_approved",
		})
	}
	return out
}

func toMOQ(qty *int32, basis string, note *string) *MOQ {
	if qty == nil {
		return nil
	}
	return &MOQ{Quantity: *qty, Basis: basis, BasisNote: note}
}

func toPrice(min, max *int32, note *string) *Price {
	if min == nil || max == nil {
		return nil
	}
	return &Price{MinIDR: *min, MaxIDR: *max, Note: note}
}

func dateString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func moqBasis(n store.NullMoqBasis) string {
	if !n.Valid {
		return ""
	}
	return string(n.MoqBasis)
}

func unavailable(id *string, kind string) *MakerUnavailable {
	msg := "maker_not_found"
	if kind == "withdrawn" {
		msg = "maker_withdrawn"
	}
	return &MakerUnavailable{
		ID:         id,
		Status:     kind,
		MessageID:  msg,
		Contact:    nil,
		SearchHref: "/",
	}
}
