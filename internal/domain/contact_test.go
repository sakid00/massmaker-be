package domain

import "testing"

func TestCanContactMakers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		role string
		want bool
	}{
		{role: "", want: true},
		{role: "artist", want: true},
		{role: "vendor", want: false},
	}
	for _, tc := range cases {
		if got := CanContactMakers(tc.role); got != tc.want {
			t.Fatalf("role %q: got %v want %v", tc.role, got, tc.want)
		}
	}
}
