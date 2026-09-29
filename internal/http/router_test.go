package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthAlwaysMounted(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health: got %d", rec.Code)
	}
}

func TestIdentityUnavailableWithoutEnmasse(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/email/status", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestInternalNotProxied(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/internal/profile-status/x", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("internal must not be on massmaker public router: got %d", rec.Code)
	}
}

func TestEventsInvalidBearerIs401(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{JWTAccessSecret: "x", JWTIssuer: "https://api.enmasse.id"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestCategoryProposalsRequireAuth(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{JWTAccessSecret: "x", JWTIssuer: "https://api.enmasse.id"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/me/category-proposals", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestMeVendorsRequiresAuth(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{JWTAccessSecret: "x", JWTIssuer: "https://api.enmasse.id"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/me/vendors", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestMeConsentsRequiresAuth(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{JWTAccessSecret: "x", JWTIssuer: "https://api.enmasse.id"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/me/consents", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestMeOrdersRequiresAuth(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{JWTAccessSecret: "x", JWTIssuer: "https://api.enmasse.id"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/me/orders", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestConsentsUnavailableWithoutService(t *testing.T) {
	t.Parallel()
	h := NewRouter(nil, nil, nil, nil, nil, nil, nil, RouterOptions{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/consents", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d", rec.Code)
	}
}
