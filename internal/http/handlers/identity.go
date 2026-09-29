package handlers

import (
	"io"
	"net/http"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/enmasse"
)

type IdentityHandler struct {
	em *enmasse.Client
}

func NewIdentityHandler(em *enmasse.Client) *IdentityHandler {
	return &IdentityHandler{em: em}
}

func (h *IdentityHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	if h.em == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, domain.ErrBadRequest)
		return
	}
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	res, err := h.em.Forward(r.Context(), r.Method, path, body, r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", res.ContentType)
	w.WriteHeader(res.Status)
	_, _ = w.Write(res.Body)
}
