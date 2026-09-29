package handlers

import (
	"net"
	"net/http"

	"github.com/google/uuid"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/service"
)

type ConsentsHandler struct {
	svc *service.Consents
}

func NewConsentsHandler(svc *service.Consents) *ConsentsHandler {
	return &ConsentsHandler{svc: svc}
}

func (h *ConsentsHandler) Put(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	var body struct {
		AnonSessionID string `json:"anonSessionId"`
		Source        string `json:"source"`
		Items         []struct {
			ConsentType string `json:"consentType"`
			Granted     bool   `json:"granted"`
		} `json:"items"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, domain.NewAppError(http.StatusBadRequest, "validation", "invalid json"))
		return
	}
	items := make([]service.PutConsentItem, 0, len(body.Items))
	for _, item := range body.Items {
		items = append(items, service.PutConsentItem{ConsentType: item.ConsentType, Granted: item.Granted})
	}
	var userID *uuid.UUID
	id, role := userFrom(r)
	if id != "" {
		parsed, err := uuid.Parse(id)
		if err == nil {
			userID = &parsed
		}
	}
	out, err := h.svc.Put(r.Context(), service.PutConsentsInput{
		AnonSessionID: body.AnonSessionID,
		Source:        body.Source,
		Items:         items,
		UserID:        userID,
		Role:          role,
		IP:            remoteIP(r),
		UserAgent:     r.UserAgent(),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"consents": out})
}

func (h *ConsentsHandler) Me(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUnavailable)
		return
	}
	id, _ := userFrom(r)
	uid, err := uuid.Parse(id)
	if err != nil {
		writeError(w, domain.ErrUnauthorized)
		return
	}
	out, err := h.svc.ListMine(r.Context(), uid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"consents": out})
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
