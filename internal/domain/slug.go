package domain

import (
	"strings"
	"unicode"
)

const (
	OtherSentinel = "__other__"
)

var reservedSlugs = map[string]struct{}{
	"other":       {},
	OtherSentinel: {},
	"all":         {},
	"categories":  {},
}

func ProposeSlug(label string) (string, error) {
	trimmed := strings.TrimSpace(label)
	if len([]rune(trimmed)) < 2 || len([]rune(trimmed)) > 80 {
		return "", NewAppError(400, "validation", "proposed name must be 2–80 characters")
	}
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(trimmed) {
		switch {
		case r == ' ' || r == '_' || r == '-':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if unicode.IsLetter(r) || unicode.IsNumber(r) {
				continue
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) < 2 || len(slug) > 40 {
		return "", NewAppError(400, "validation", "could not make a category slug from that name")
	}
	if _, reserved := reservedSlugs[slug]; reserved {
		return "", NewAppError(400, "validation", "that category name is reserved")
	}
	return slug, nil
}

func CapabilityAllowed(status string, proposedBy *string, makerID string) bool {
	if status == "published" {
		return true
	}
	return status == "draft" && proposedBy != nil && *proposedBy == makerID
}
