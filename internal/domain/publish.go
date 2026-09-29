package domain

import "net/http"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusWithdrawn = "withdrawn"
)

func PublicMakerCode(status string, demo bool) int {
	if demo {
		return http.StatusNotFound
	}
	switch status {
	case StatusPublished:
		return http.StatusOK
	case StatusWithdrawn:
		return http.StatusGone
	default:
		return http.StatusNotFound
	}
}
