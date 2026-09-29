package domain_test

import (
	"testing"

	"github.com/sakid00/massmaker-be/internal/domain"
)

func TestNormalizeQuery(t *testing.T) {
	t.Parallel()
	if got := domain.NormalizeQuery("  Stiker   PIN "); got != "stiker pin" {
		t.Fatalf("got %q", got)
	}
}

func TestHasPIIKeys(t *testing.T) {
	t.Parallel()
	if !domain.HasPIIKeys(map[string]any{"email": "x"}) {
		t.Fatal("expected pii")
	}
	if !domain.HasPIIKeys(map[string]any{"search": map[string]any{"email": "x"}}) {
		t.Fatal("nested email is pii")
	}
	if domain.HasPIIKeys(map[string]any{"type": "search"}) {
		t.Fatal("type is not pii")
	}
}
