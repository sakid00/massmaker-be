package service

import (
	"testing"

	"github.com/sakid00/massmaker-be/internal/enmasse"
)

func TestVendorIdentityReady(t *testing.T) {
	t.Parallel()
	if vendorIdentityReady(nil) {
		t.Fatal("nil vendor must be incomplete")
	}
	if vendorIdentityReady(&enmasse.InternalVendor{Name: "Studio", City: "Bandung"}) {
		t.Fatal("signup-only vendor must be incomplete")
	}
	full := &enmasse.InternalVendor{
		Name:     "Studio",
		Province: "Jawa Barat",
		City:     "Kota Bandung",
		District: "Sumur Bandung",
		PICName:  "Ayu",
		Address:  "Jl. Merdeka 1",
		WhatsApp: "+628111111111",
		Bio:      "Sablon kaos booth",
	}
	if !vendorIdentityReady(full) {
		t.Fatal("expected complete vendor identity")
	}
}
