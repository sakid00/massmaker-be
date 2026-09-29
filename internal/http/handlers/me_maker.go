package handlers

import (
	"net/http"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/service"
)

type MeMakerHandler struct {
	svc *service.MakerSelf
}

func NewMeMakerHandler(svc *service.MakerSelf) *MeMakerHandler {
	return &MeMakerHandler{svc: svc}
}

func (h *MeMakerHandler) Onboarding(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	out, err := h.svc.Onboarding(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *MeMakerHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	out, err := h.svc.Get(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *MeMakerHandler) Patch(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var body struct {
		Capabilities     []string                 `json:"capabilities"`
		MOQ              *service.MOQ             `json:"moq"`
		LeadTimeEstimate *string                  `json:"leadTimeEstimate"`
		PriceMinIDR      *int32                   `json:"priceMinIdr"`
		PriceMaxIDR      *int32                   `json:"priceMaxIdr"`
		PriceNote        *string                  `json:"priceNote"`
		HoursOpen        *string                  `json:"hoursOpen"`
		HoursClose       *string                  `json:"hoursClose"`
		HoursDays        *[]string                `json:"hoursDays"`
		Portfolio        *[]service.PortfolioItem `json:"portfolio"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	in := service.PatchMakerInput{
		Capabilities:     body.Capabilities,
		MOQ:              body.MOQ,
		LeadTimeEstimate: body.LeadTimeEstimate,
		PriceMinIDR:      body.PriceMinIDR,
		PriceMaxIDR:      body.PriceMaxIDR,
		PriceNote:        body.PriceNote,
		HoursOpen:        body.HoursOpen,
		HoursClose:       body.HoursClose,
		Portfolio:        body.Portfolio,
	}
	if body.HoursDays != nil {
		in.HoursDays = *body.HoursDays
		in.HoursDaysSet = true
	}
	id, _ := userFrom(r)
	out, err := h.svc.Patch(r.Context(), r.Header.Get("Authorization"), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *MeMakerHandler) ProposeCategory(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var body struct {
		Kind      string  `json:"kind"`
		LabelID   string  `json:"labelId"`
		GroupSlug *string `json:"groupSlug"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	group := ""
	if body.GroupSlug != nil {
		group = *body.GroupSlug
	}
	id, _ := userFrom(r)
	out, err := h.svc.Propose(r.Context(), r.Header.Get("Authorization"), id, body.Kind, body.LabelID, group)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *MeMakerHandler) ListCategoryProposals(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	out, err := h.svc.ListProposals(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
