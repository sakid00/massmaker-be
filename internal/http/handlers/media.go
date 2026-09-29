package handlers

import (
	"net/http"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/media"
)

type MediaHandler struct {
	presigner *media.Presigner
}

func NewMediaHandler(presigner *media.Presigner) *MediaHandler {
	return &MediaHandler{presigner: presigner}
}

func (h *MediaHandler) Presign(w http.ResponseWriter, r *http.Request) {
	if h.presigner == nil || !h.presigner.Configured() {
		writeError(w, domain.ErrMediaUnavailable)
		return
	}
	userID, _ := userFrom(r)
	if userID == "" {
		writeError(w, domain.ErrUnauthorized)
		return
	}
	var body struct {
		ContentType   string `json:"contentType"`
		ContentLength int64  `json:"contentLength"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	out, err := h.presigner.Presign(r.Context(), userID, body.ContentType, body.ContentLength)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
