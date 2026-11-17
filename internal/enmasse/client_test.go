package enmasse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sakid00/massmaker-be/internal/domain"
	"github.com/sakid00/massmaker-be/internal/enmasse"
)

func TestMissingURLFailsClosed(t *testing.T) {
	t.Parallel()
	c := enmasse.New("", "secret-secret-secret-secret-secret")
	_, err := c.ProfileStatus(context.Background(), "u1")
	if err != domain.ErrUnavailable {
		t.Fatalf("got %v", err)
	}
}

func TestProfileStatusOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing service jwt")
		}
		if r.URL.Path != "/v1/internal/profile-status/user-1" {
			t.Errorf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"complete": true, "missing": []string{}, "emailVerified": true})
	}))
	defer srv.Close()

	c := enmasse.New(srv.URL, "secret-secret-secret-secret-secret")
	got, err := c.ProfileStatus(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Complete {
		t.Fatal("expected complete")
	}
	if !got.EmailIsVerified() {
		t.Fatal("expected verified")
	}
}

func TestProfileStatusMissingEmailVerifiedIsGrandfathered(t *testing.T) {
	t.Parallel()
	st := enmasse.ProfileStatus{Complete: true, Missing: []string{}}
	if !st.EmailIsVerified() {
		t.Fatal("missing emailVerified must not block overlay edits")
	}
	unverified := false
	st.EmailVerified = &unverified
	if st.EmailIsVerified() {
		t.Fatal("explicit false must block")
	}
}

func TestForwardUsesMassmakerPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/email/status" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer user-token" {
			t.Errorf("auth %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Forwarded-For") != "203.0.113.10" {
			t.Errorf("xff %s", r.Header.Get("X-Forwarded-For"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"available"}`))
	}))
	defer srv.Close()

	c := enmasse.New(srv.URL, "secret-secret-secret-secret-secret")
	got, err := c.Forward(context.Background(), http.MethodPost, "/v1/auth/email/status", []byte(`{"email":"a@b.co"}`), "Bearer user-token", "203.0.113.10")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != http.StatusOK {
		t.Fatalf("status %d", got.Status)
	}
}

func TestForwardMissingURLFailsClosed(t *testing.T) {
	t.Parallel()
	c := enmasse.New("", "secret-secret-secret-secret-secret")
	_, err := c.Forward(context.Background(), http.MethodPost, "/v1/auth/login", nil, "", "")
	if err != domain.ErrUnavailable {
		t.Fatalf("got %v", err)
	}
}

func TestEnmasseDownIs503(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := enmasse.New(srv.URL, "secret-secret-secret-secret-secret")
	_, err := c.ProfileStatus(context.Background(), "user-1")
	if err != domain.ErrUnavailable {
		t.Fatalf("got %v", err)
	}
}
