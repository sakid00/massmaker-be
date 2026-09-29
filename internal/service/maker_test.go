package service

import "testing"

func TestOverlayReady(t *testing.T) {
	t.Parallel()
	min, max := int32(10000), int32(50000)
	qty := int32(24)
	days := []string{"mon", "tue", "wed", "thu", "fri"}
	if OverlayReady(&min, &max, "09:00", "17:00", days, &qty, "per_desain", "10 hari", 1) != true {
		t.Fatal("expected ready without portfolio")
	}
	if OverlayReady(&min, &max, "09:00", "17:00", days, &qty, "per_desain", "10 hari", 0) {
		t.Fatal("need a category")
	}
	if OverlayReady(nil, &max, "09:00", "17:00", days, &qty, "per_desain", "10 hari", 1) {
		t.Fatal("need price")
	}
	if OverlayReady(&min, &max, "09:00", "17:00", nil, &qty, "per_desain", "10 hari", 1) {
		t.Fatal("need days")
	}
	if OverlayReady(&min, &max, "09:00", "17:00", days, nil, "per_desain", "10 hari", 1) {
		t.Fatal("need moq")
	}
	zero := int32(0)
	if OverlayReady(&min, &max, "09:00", "17:00", days, &zero, "per_desain", "10 hari", 1) {
		t.Fatal("need positive moq")
	}
	if OverlayReady(&min, &max, "09:00", "17:00", days, &qty, "unknown", "10 hari", 1) {
		t.Fatal("need known moq basis")
	}
	if OverlayReady(&min, &max, "09:00", "17:00", days, &qty, "per_desain", "", 1) {
		t.Fatal("need lead time")
	}
}

func TestParseMoqBasis(t *testing.T) {
	t.Parallel()
	empty, err := parseMoqBasis("  ")
	if err != nil || empty.Valid {
		t.Fatal("empty basis is unset")
	}
	got, err := parseMoqBasis("total")
	if err != nil || !got.Valid || string(got.MoqBasis) != "total" {
		t.Fatal("expected total")
	}
	if _, err := parseMoqBasis("per-booth"); err == nil {
		t.Fatal("unknown basis")
	}
}
