package domain

import "strings"

var weekDays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

// NormalizeHoursDays keeps unique weekday tokens in Sunday-last week order.
func NormalizeHoursDays(in []string) ([]string, error) {
	allowed := map[string]struct{}{}
	for _, day := range weekDays {
		allowed[day] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, raw := range in {
		day := strings.ToLower(strings.TrimSpace(raw))
		if day == "" {
			continue
		}
		if _, ok := allowed[day]; !ok {
			return nil, NewAppError(400, "validation", "hoursDays must be mon-sun")
		}
		seen[day] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for _, day := range weekDays {
		if _, ok := seen[day]; ok {
			out = append(out, day)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}
