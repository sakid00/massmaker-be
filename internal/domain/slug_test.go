package domain_test

import (
	"testing"

	"github.com/sakid00/massmaker-be/internal/domain"
)

func TestProposeSlug(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"sablon DTF", "sablon-dtf", false},
		{"  Tote_Bag  ", "tote-bag", false},
		{"3D Print", "3d-print", false},
		{"a", "", true},
		{"other", "", true},
		{"__other__", "", true},
		{"all", "", true},
		{"categories", "", true},
		{"!!!", "", true},
	}
	for _, tc := range cases {
		got, err := domain.ProposeSlug(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCapabilityAllowed(t *testing.T) {
	t.Parallel()
	maker := "maker-1"
	other := "maker-2"
	if !domain.CapabilityAllowed("published", nil, maker) {
		t.Fatal("published is allowed")
	}
	if !domain.CapabilityAllowed("draft", &maker, maker) {
		t.Fatal("own draft is allowed")
	}
	if domain.CapabilityAllowed("draft", &other, maker) {
		t.Fatal("someone else's draft is not allowed")
	}
	if domain.CapabilityAllowed("withdrawn", &maker, maker) {
		t.Fatal("withdrawn is not allowed")
	}
}
