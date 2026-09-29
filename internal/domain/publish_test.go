package domain_test

import (
	"net/http"
	"testing"

	"github.com/sakid00/massmaker-be/internal/domain"
)

func TestPublicMakerCode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status string
		demo   bool
		want   int
	}{
		{domain.StatusPublished, false, http.StatusOK},
		{domain.StatusWithdrawn, false, http.StatusGone},
		{domain.StatusDraft, false, http.StatusNotFound},
		{domain.StatusPublished, true, http.StatusNotFound},
	}
	for _, tc := range cases {
		if got := domain.PublicMakerCode(tc.status, tc.demo); got != tc.want {
			t.Errorf("%s demo=%v: got %d want %d", tc.status, tc.demo, got, tc.want)
		}
	}
}
