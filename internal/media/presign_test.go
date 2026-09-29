package media

import (
	"strings"
	"testing"
)

func TestExtForContentType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		ext  string
		fail bool
	}{
		{in: "image/jpeg", ext: "jpg"},
		{in: " image/PNG ", ext: "png"},
		{in: "image/webp", ext: "webp"},
		{in: "image/gif", fail: true},
		{in: "application/pdf", fail: true},
		{in: "", fail: true},
	}
	for _, tc := range cases {
		ext, err := ExtForContentType(tc.in)
		if tc.fail {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil || ext != tc.ext {
			t.Fatalf("%q: got %q %v want %q", tc.in, ext, err, tc.ext)
		}
	}
}

func TestObjectKey(t *testing.T) {
	t.Parallel()
	key := ObjectKey("user-1", "jpg")
	if !strings.HasPrefix(key, "profiles/user-1/") || !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("key %q", key)
	}
}

func TestPublicURL(t *testing.T) {
	t.Parallel()
	got := PublicURL("https://cdn.example/", "profiles/a/b.jpg")
	if got != "https://cdn.example/profiles/a/b.jpg" {
		t.Fatalf("got %q", got)
	}
}

func TestConfigComplete(t *testing.T) {
	t.Parallel()
	empty := Config{}
	if empty.Complete() {
		t.Fatal("empty should be incomplete")
	}
	full := Config{
		AccountID:       "acct",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Bucket:          "bucket",
		PublicBaseURL:   "https://cdn.example",
	}
	if !full.Complete() {
		t.Fatal("full should be complete")
	}
}

func TestValidateContentLength(t *testing.T) {
	t.Parallel()
	const twoMiB = 2 * 1024 * 1024
	cases := []struct {
		n    int64
		fail bool
	}{
		{n: 0, fail: true},
		{n: -1, fail: true},
		{n: 1, fail: false},
		{n: twoMiB, fail: false},
		{n: twoMiB + 1, fail: true},
	}
	for _, tc := range cases {
		err := ValidateContentLength(tc.n)
		if tc.fail {
			if err == nil {
				t.Fatalf("%d: expected error", tc.n)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%d: %v", tc.n, err)
		}
	}
}

func TestInquiryExtForContentType(t *testing.T) {
	t.Parallel()
	ext, err := InquiryExtForContentType("application/pdf")
	if err != nil || ext != "pdf" {
		t.Fatalf("pdf: %q %v", ext, err)
	}
	if _, err := InquiryExtForContentType("image/gif"); err == nil {
		t.Fatal("gif should fail")
	}
}

func TestInquiryURLOwned(t *testing.T) {
	t.Parallel()
	base := "https://cdn.example"
	user := "user-1"
	ok := InquiryURLOwned("https://cdn.example/inquiries/user-1/a.pdf", base, user)
	if !ok {
		t.Fatal("expected owned")
	}
	if InquiryURLOwned("https://cdn.example/profiles/user-1/a.jpg", base, user) {
		t.Fatal("profile url is not inquiry")
	}
	if InquiryURLOwned("https://cdn.example/inquiries/other/a.pdf", base, user) {
		t.Fatal("other user")
	}
	if !InquiryURLOwned("https://cdn.example/inquiries/user-1/a.pdf?x=1", "", user) {
		t.Fatal("path ownership should not need public base")
	}
}

func TestUnconfiguredPresigner(t *testing.T) {
	t.Parallel()
	p := &Presigner{}
	if p.Configured() {
		t.Fatal("expected unconfigured")
	}
	_, err := p.Presign(t.Context(), "user-1", "image/jpeg", 1024)
	if err == nil {
		t.Fatal("expected unavailable")
	}
}
