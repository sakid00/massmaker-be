package domain

import (
	"strings"
	"unicode"
)

func NormalizeQuery(q string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(q)))
	return strings.Join(fields, " ")
}

func HasPIIKeys(v any) bool {
	switch body := v.(type) {
	case map[string]any:
		for k, child := range body {
			switch strings.ToLower(k) {
			case "phone", "email", "whatsapp":
				return true
			}
			if HasPIIKeys(child) {
				return true
			}
		}
	case []any:
		for _, child := range body {
			if HasPIIKeys(child) {
				return true
			}
		}
	}
	return false
}

func NeedsReconfirmation(verificationYYYYMMDD string, nowYear, nowMonth, nowDay int) bool {
	if len(verificationYYYYMMDD) < 10 {
		return true
	}
	var y, m, d int
	if _, err := parseYMD(verificationYYYYMMDD, &y, &m, &d); err != nil {
		return true
	}
	// older than 365 days ≈ previous calendar year same day
	if nowYear-y > 1 {
		return true
	}
	if nowYear-y == 1 {
		if nowMonth > m || (nowMonth == m && nowDay >= d) {
			return true
		}
	}
	return false
}

func parseYMD(s string, y, m, d *int) (int, error) {
	if len(s) < 10 || s[4] != '-' || s[7] != '-' {
		return 0, errYMD
	}
	*y = atoi3(s[0:4])
	*m = atoi3(s[5:7])
	*d = atoi3(s[8:10])
	if *y == 0 || *m == 0 || *d == 0 {
		return 0, errYMD
	}
	return 0, nil
}

var errYMD = errParse("ymd")

type errParse string

func (e errParse) Error() string { return string(e) }

func atoi3(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
