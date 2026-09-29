package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/service"
)

type OrdersHandler struct {
	svc *service.Orders
}

func NewOrdersHandler(svc *service.Orders) *OrdersHandler {
	return &OrdersHandler{svc: svc}
}

func (h *OrdersHandler) Presign(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	userID, role := userFrom(r)
	var body struct {
		ContentType   string `json:"contentType"`
		ContentLength int64  `json:"contentLength"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	out, err := h.svc.Presign(r.Context(), userID, role, body.ContentType, body.ContentLength)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *OrdersHandler) Context(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	userID, role := userFrom(r)
	out, code, err := h.svc.Context(r.Context(), r.Header.Get("Authorization"), userID, role, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if out == nil {
		writeError(w, statusError(code))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *OrdersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var body service.CreateInquiryInput
	if err := decodeInquiry(r, &body); err != nil {
		slog.Error("inquiry json", "err", err)
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	userID, role := userFrom(r)
	out, err := h.svc.Create(r.Context(), r.Header.Get("Authorization"), userID, role, chi.URLParam(r, "id"), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *OrdersHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	userID, role := userFrom(r)
	out, err := h.svc.List(r.Context(), r.Header.Get("Authorization"), userID, role)
	if err != nil {
		writeError(w, err)
		return
	}
	if out == nil {
		out = []service.OrderCard{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": out})
}

func (h *OrdersHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	userID, role := userFrom(r)
	out, err := h.svc.Get(r.Context(), r.Header.Get("Authorization"), userID, role, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *OrdersHandler) Patch(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var body service.PatchOrderInput
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	userID, role := userFrom(r)
	out, err := h.svc.Patch(r.Context(), r.Header.Get("Authorization"), userID, role, chi.URLParam(r, "id"), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func statusError(code int) error {
	switch code {
	case http.StatusGone:
		return domain.ErrGone
	case http.StatusNotFound:
		return domain.ErrNotFound
	default:
		return domain.ErrNotFound
	}
}
