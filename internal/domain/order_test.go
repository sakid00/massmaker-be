package domain

import (
	"testing"
	"time"
)

func TestCanCreateInquiry(t *testing.T) {
	t.Parallel()
	if CanCreateInquiry("vendor") || CanCreateInquiry("") || !CanCreateInquiry("artist") {
		t.Fatal("only artists create inquiries")
	}
}

func TestApplyOrderStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		role, from, to string
		ok             bool
	}{
		{role: "vendor", from: OrderPendingReview, to: OrderAccepted, ok: true},
		{role: "vendor", from: OrderPendingReview, to: OrderDeclined, ok: true},
		{role: "artist", from: OrderPendingReview, to: OrderCancelled, ok: true},
		{role: "artist", from: OrderPendingReview, to: OrderAccepted, ok: false},
		{role: "vendor", from: OrderPendingReview, to: OrderCancelled, ok: false},
		{role: "vendor", from: OrderAccepted, to: OrderDeclined, ok: false},
		{role: "artist", from: OrderDeclined, to: OrderCancelled, ok: false},
	}
	for _, tc := range cases {
		err := ApplyOrderStatus(tc.role, tc.from, tc.to)
		if tc.ok && err != nil {
			t.Fatalf("%s %s -> %s: %v", tc.role, tc.from, tc.to, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%s %s -> %s: expected error", tc.role, tc.from, tc.to)
		}
	}
}

func TestValidateOrderDates(t *testing.T) {
	t.Parallel()
	today := time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)
	ready, err := ParseOrderDate("2026-09-29")
	if err != nil {
		t.Fatal(err)
	}
	ship, err := ParseOrderDate("2026-10-20")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateOrderDates(ready, ship, today); err != nil {
		t.Fatal(err)
	}
	past, _ := ParseOrderDate("2026-09-28")
	if err := ValidateOrderDates(past, ship, today); err == nil {
		t.Fatal("past ready date")
	}
	earlyShip, _ := ParseOrderDate("2026-09-20")
	if err := ValidateOrderDates(ready, earlyShip, today); err == nil {
		t.Fatal("ship before ready")
	}
}
