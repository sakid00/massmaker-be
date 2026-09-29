package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sakid00/massmaker-be/internal/domain"
)

type errorBody struct {
	Error *domain.AppError `json:"error"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("respond: encode json", "err", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	if ae, ok := domain.IsAppError(err); ok {
		writeJSON(w, ae.Status, errorBody{Error: ae})
		return
	}
	slog.Error("internal error", "err", err)
	writeJSON(w, http.StatusInternalServerError, errorBody{
		Error: domain.NewAppError(http.StatusInternalServerError, "internal", "internal server error"),
	})
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
