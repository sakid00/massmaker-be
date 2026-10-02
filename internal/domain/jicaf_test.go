package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeJicafCollaboration_False(t *testing.T) {
	artistIn := "Jane Artist"
	y := int32(2024)
	before, artist, year, err := NormalizeJicafCollaboration(false, &artistIn, &y)
	if err != nil {
		t.Fatal(err)
	}
	if before {
		t.Fatal("expected false")
	}
	if artist != nil || year != nil {
		t.Fatalf("expected nil artist/year, got %v %v", artist, year)
	}
	before, artist, year, err = NormalizeJicafCollaboration(false, nil, nil)
	if err != nil || before || artist != nil || year != nil {
		t.Fatalf("false empty: got before=%v artist=%v year=%v err=%v", before, artist, year, err)
	}
}

func TestNormalizeJicafCollaboration_True(t *testing.T) {
	artistIn := "  Jane Artist  "
	y := int32(2024)
	before, artist, year, err := NormalizeJicafCollaboration(true, &artistIn, &y)
	if err != nil {
		t.Fatal(err)
	}
	if !before {
		t.Fatal("expected true")
	}
	if artist == nil || *artist != "Jane Artist" {
		t.Fatalf("artist: %v", artist)
	}
	if year == nil || *year != 2024 {
		t.Fatalf("year: %v", year)
	}

	if _, _, _, err := NormalizeJicafCollaboration(true, nil, &y); err == nil {
		t.Fatal("expected missing artist")
	}
	if _, _, _, err := NormalizeJicafCollaboration(true, &artistIn, nil); err == nil {
		t.Fatal("expected missing year")
	}
	short := "J"
	if _, _, _, err := NormalizeJicafCollaboration(true, &short, &y); err == nil {
		t.Fatal("expected short artist")
	}
	longStr := strings.Repeat("a", 81)
	if _, _, _, err := NormalizeJicafCollaboration(true, &longStr, &y); err == nil {
		t.Fatal("expected long artist")
	}
	old := int32(JicafCollaborationYearMin - 1)
	if _, _, _, err := NormalizeJicafCollaboration(true, &artistIn, &old); err == nil {
		t.Fatal("expected year too old")
	}
	future := int32(time.Now().Year() + 1)
	if _, _, _, err := NormalizeJicafCollaboration(true, &artistIn, &future); err == nil {
		t.Fatal("expected year in future")
	}
	current := int32(time.Now().Year())
	if _, _, _, err := NormalizeJicafCollaboration(true, &artistIn, &current); err != nil {
		t.Fatalf("current year should be ok: %v", err)
	}
	minY := int32(JicafCollaborationYearMin)
	if _, _, _, err := NormalizeJicafCollaboration(true, &artistIn, &minY); err != nil {
		t.Fatalf("min year should be ok: %v", err)
	}
}
