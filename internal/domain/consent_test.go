package domain_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/sakid00/massmaker-be/internal/domain"
)

func TestConsentTypeAllowed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		role, typ string
		want      bool
	}{
		{"", domain.ConsentActivityTracking, true},
		{"", domain.ConsentMarketing, true},
		{"artist", domain.ConsentActivityTracking, true},
		{"", domain.ConsentPrivacyPolicy, false},
		{"artist", domain.ConsentPrivacyPolicy, true},
		{"vendor", domain.ConsentTerms, true},
		{"", domain.ConsentWhatsAppPublication, false},
		{"artist", domain.ConsentWhatsAppPublication, false},
		{"vendor", domain.ConsentWhatsAppPublication, true},
		{"vendor", domain.ConsentRecommendationPublication, false},
		{"artist", domain.ConsentRecommendationPublication, true},
		{"artist", "unknown", false},
	}
	for _, tc := range cases {
		if got := domain.ConsentTypeAllowed(tc.role, tc.typ); got != tc.want {
			t.Errorf("role=%q type=%q: got %v want %v", tc.role, tc.typ, got, tc.want)
		}
	}
}

func TestShowPublicWhatsApp(t *testing.T) {
	t.Parallel()
	uid := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if domain.ShowPublicWhatsApp(false, nil, true) {
		t.Fatal("staff permit is required")
	}
	if !domain.ShowPublicWhatsApp(true, nil, false) {
		t.Fatal("legacy makers without a bound user keep staff-only permit")
	}
	if domain.ShowPublicWhatsApp(true, &uid, false) {
		t.Fatal("bound vendor needs publication grant")
	}
	if !domain.ShowPublicWhatsApp(true, &uid, true) {
		t.Fatal("staff permit and vendor grant")
	}
}
