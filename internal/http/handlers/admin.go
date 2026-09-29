package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/service"
)

type AdminHandler struct {
	svc *service.Admin
}

func NewAdminHandler(svc *service.Admin) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	tok, err := h.svc.Login(body.Email, body.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"accessToken": tok})
}

func (h *AdminHandler) ListMakers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.ListMakers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"makers": rows})
}

func (h *AdminHandler) CreateMaker(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name             string   `json:"name"`
		City             string   `json:"city"`
		VerificationDate string   `json:"verificationDate"`
		Capabilities     []string `json:"capabilities"`
		EnmasseVendorID  *string  `json:"enmasseVendorId"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	var vendor *uuid.UUID
	if body.EnmasseVendorID != nil && *body.EnmasseVendorID != "" {
		id, err := uuid.Parse(*body.EnmasseVendorID)
		if err != nil {
			writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid enmasseVendorId"))
			return
		}
		vendor = &id
	}
	id, err := h.svc.CreateMaker(r.Context(), service.CreateMakerInput{
		Name:             body.Name,
		City:             body.City,
		VerificationDate: body.VerificationDate,
		Capabilities:     body.Capabilities,
		EnmasseVendorID:  vendor,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *AdminHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	if err := h.svc.Publish(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AdminHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	if err := h.svc.Withdraw(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AdminHandler) PutContact(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	var body struct {
		Channel    string `json:"channel"`
		E164Digits string `json:"e164Digits"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	if err := h.svc.PutContact(r.Context(), id, body.Channel, body.E164Digits); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AdminHandler) Consent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	if err := h.svc.ConsentContact(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AdminHandler) ListTaxonomyDrafts(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListTaxonomyDrafts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) PublishCategoryGroup(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetGroupPublication(r.Context(), chi.URLParam(r, "slug"), "published"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AdminHandler) WithdrawCategoryGroup(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetGroupPublication(r.Context(), chi.URLParam(r, "slug"), "withdrawn"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AdminHandler) PublishCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetLeafPublication(r.Context(), chi.URLParam(r, "slug"), "published"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AdminHandler) WithdrawCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetLeafPublication(r.Context(), chi.URLParam(r, "slug"), "withdrawn"); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
