package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/service"
)

type PublicHandler struct {
	cat    *service.Catalogue
	events *service.Events
}

func NewPublicHandler(cat *service.Catalogue, events *service.Events) *PublicHandler {
	return &PublicHandler{cat: cat, events: events}
}

func (h *PublicHandler) Categories(w http.ResponseWriter, r *http.Request) {
	out, err := h.cat.Categories(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *PublicHandler) Search(w http.ResponseWriter, r *http.Request) {
	out, err := h.cat.Search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("category"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *PublicHandler) Maker(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	detail, unavail, code, err := h.cat.Maker(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if unavail != nil {
		writeJSON(w, code, unavail)
		return
	}
	writeJSON(w, code, detail)
}

func (h *PublicHandler) Events(w http.ResponseWriter, r *http.Request) {
	if h.events == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, domain.ErrBadRequest)
		return
	}
	var userID *uuid.UUID
	id, role := userFrom(r)
	if id != "" {
		parsed, err := uuid.Parse(id)
		if err == nil {
			userID = &parsed
		}
	}
	items, err := h.events.Ingest(r.Context(), json.RawMessage(raw), userID, role)
	if err != nil {
		writeError(w, err)
		return
	}
	counted := len(items) > 0 && items[0].Counted
	writeJSON(w, http.StatusOK, map[string]any{"accepted": true, "counted": counted, "events": items})
}

func (h *PublicHandler) MeVendors(w http.ResponseWriter, r *http.Request) {
	id, role := userFrom(r)
	cards, err := h.cat.MeVendors(r.Context(), id, role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"makers": cards})
}
