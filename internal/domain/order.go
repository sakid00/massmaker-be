package domain

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	OrderPendingReview = "pending_review"
	OrderAccepted      = "accepted"
	OrderDeclined      = "declined"
	OrderCancelled     = "cancelled"

	MaxOrderAttachments = 8
	MinNameRunes        = 2
	MaxNameRunes        = 80
	MaxNoteRunes        = 500
)

func CanCreateInquiry(role string) bool {
	return role == "artist"
}

func CanListOrders(role string) bool {
	return role == "artist" || role == "vendor"
}

func ApplyOrderStatus(role, from, to string) error {
	to = strings.TrimSpace(to)
	from = strings.TrimSpace(from)
	switch role {
	case "artist":
		if from == OrderPendingReview && to == OrderCancelled {
			return nil
		}
	case "vendor":
		if from == OrderPendingReview && (to == OrderAccepted || to == OrderDeclined) {
			return nil
		}
	}
	return NewAppError(http.StatusBadRequest, "validation", "invalid status transition")
}

func ClampOrderName(raw, field string) (string, error) {
	s := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(s)
	if n < MinNameRunes {
		return "", NewAppError(http.StatusBadRequest, "validation", field+" is required")
	}
	if n > MaxNameRunes {
		return "", NewAppError(http.StatusBadRequest, "validation", field+" is too long")
	}
	return s, nil
}

func ClampOrderNote(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) > MaxNoteRunes {
		return "", NewAppError(http.StatusBadRequest, "validation", "note is too long")
	}
	return s, nil
}

func ParseOrderDate(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, NewAppError(http.StatusBadRequest, "validation", "invalid date")
	}
	return t, nil
}

func ValidateOrderDates(ready, ship, today time.Time) error {
	readyDay := ready.UTC().Truncate(24 * time.Hour)
	shipDay := ship.UTC().Truncate(24 * time.Hour)
	todayDay := today.UTC().Truncate(24 * time.Hour)
	if readyDay.Before(todayDay) {
		return NewAppError(http.StatusBadRequest, "validation", "ready date must be today or later")
	}
	if shipDay.Before(readyDay) {
		return NewAppError(http.StatusBadRequest, "validation", "ship date must be on or after ready date")
	}
	return nil
}
