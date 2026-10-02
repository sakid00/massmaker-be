package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeJicafCollaboration_Unset(t *testing.T) {
	before, artist, year, err := NormalizeJicafCollaboration(nil, nil, nil)
	if err != nil || before != nil || artist != nil || year != nil {
		t.Fatalf("unset: got before=%v artist=%v year=%v err=%v", before, artist, year, err)
	}
	artistIn := "Jane"
	y := int32(2024)
	if _, _, _, err := NormalizeJicafCollaboration(nil, &artistIn, nil); err == nil {
		t.Fatal("expected error when artist set without before")
	}
	if _, _, _, err := NormalizeJicafCollaboration(nil, nil, &y); err == nil {
		t.Fatal("expected error when year set without before")
	}
}

func TestNormalizeJicafCollaboration_False(t *testing.T) {
	f := false
	artistIn := "Jane Artist"
	y := int32(2024)
	before, artist, year, err := NormalizeJicafCollaboration(&f, &artistIn, &y)
	if err != nil {
		t.Fatal(err)
	}
	if before == nil || *before {
		t.Fatalf("expected false, got %v", before)
	}
	if artist != nil || year != nil {
		t.Fatalf("expected nil artist/year, got %v %v", artist, year)
	}
}

func TestNormalizeJicafCollaboration_True(t *testing.T) {
	tr := true
	artistIn := "  Jane Artist  "
	y := int32(2024)
	before, artist, year, err := NormalizeJicafCollaboration(&tr, &artistIn, &y)
	if err != nil {
		t.Fatal(err)
	}
	if before == nil || !*before {
		t.Fatal("expected true")
	}
	if artist == nil || *artist != "Jane Artist" {
		t.Fatalf("artist: %v", artist)
	}
	if year == nil || *year != 2024 {
		t.Fatalf("year: %v", year)
	}

	if _, _, _, err := NormalizeJicafCollaboration(&tr, nil, &y); err == nil {
		t.Fatal("expected missing artist")
	}
	if _, _, _, err := NormalizeJicafCollaboration(&tr, &artistIn, nil); err == nil {
		t.Fatal("expected missing year")
	}
	short := "J"
	if _, _, _, err := NormalizeJicafCollaboration(&tr, &short, &y); err == nil {
		t.Fatal("expected short artist")
	}
	longStr := strings.Repeat("a", 81)
	if _, _, _, err := NormalizeJicafCollaboration(&tr, &longStr, &y); err == nil {
		t.Fatal("expected long artist")
	}
	old := int32(JicafCollaborationYearMin - 1)
	if _, _, _, err := NormalizeJicafCollaboration(&tr, &artistIn, &old); err == nil {
		t.Fatal("expected year too old")
	}
	future := int32(time.Now().Year() + 1)
	if _, _, _, err := NormalizeJicafCollaboration(&tr, &artistIn, &future); err == nil {
		t.Fatal("expected year in future")
	}
	current := int32(time.Now().Year())
	if _, _, _, err := NormalizeJicafCollaboration(&tr, &artistIn, &current); err != nil {
		t.Fatalf("current year should be ok: %v", err)
	}
	minY := int32(JicafCollaborationYearMin)
	if _, _, _, err := NormalizeJicafCollaboration(&tr, &artistIn, &minY); err != nil {
		t.Fatalf("min year should be ok: %v", err)
	}
}
