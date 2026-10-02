package domain

import (
	"strings"
	"time"
)

const (
	JicafCollaborationYearMin = 2015
	jicafArtistMinRunes       = 2
	jicafArtistMaxRunes       = 80
)

// NormalizeJicafCollaboration validates and normalizes JICAF collaboration fields.
// When collaboratedBefore is false, artist and year are forced to nil.
// When true, artist (trimmed, 2–80 runes) and year (2015–current) are required.
// When collaboratedBefore is nil, artist and year must also be nil.
func NormalizeJicafCollaboration(collaboratedBefore *bool, artist *string, year *int32) (*bool, *string, *int32, error) {
	if collaboratedBefore == nil {
		if artist != nil || year != nil {
			return nil, nil, nil, NewAppError(400, "validation", "jicafCollaboratedBefore is required when collaboration details are set")
		}
		return nil, nil, nil, nil
	}
	if !*collaboratedBefore {
		return collaboratedBefore, nil, nil, nil
	}
	if artist == nil {
		return nil, nil, nil, NewAppError(400, "validation", "jicafCollaborationArtist is required")
	}
	trimmed := strings.TrimSpace(*artist)
	n := len([]rune(trimmed))
	if n < jicafArtistMinRunes || n > jicafArtistMaxRunes {
		return nil, nil, nil, NewAppError(400, "validation", "jicafCollaborationArtist must be 2–80 characters")
	}
	if year == nil {
		return nil, nil, nil, NewAppError(400, "validation", "jicafCollaborationYear is required")
	}
	maxYear := int32(time.Now().Year())
	if *year < JicafCollaborationYearMin || *year > maxYear {
		return nil, nil, nil, NewAppError(400, "validation", "jicafCollaborationYear is out of range")
	}
	return collaboratedBefore, &trimmed, year, nil
}
