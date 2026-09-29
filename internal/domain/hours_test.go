package domain

import "testing"

func TestNormalizeHoursDays(t *testing.T) {
	t.Parallel()
	got, err := NormalizeHoursDays([]string{"fri", "MON", "fri", "wed"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"mon", "wed", "fri"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if _, err := NormalizeHoursDays([]string{"monday"}); err == nil {
		t.Fatal("expected invalid day")
	}
}
