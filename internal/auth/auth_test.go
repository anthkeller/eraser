package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeVerifier struct {
	principal Principal
	err       error
}

func (v fakeVerifier) Verify(context.Context, string) (Principal, error) {
	return v.principal, v.err
}

func TestMiddlewareEnforcesBearerTokenAndOwner(t *testing.T) {
	middleware := Middleware(fakeVerifier{principal: Principal{Subject: "owner-1"}}, "owner-1", nil)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok || principal.Subject != "owner-1" {
			t.Fatal("missing verified principal")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token returned %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "Bearer signed-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid token returned %d", response.Code)
	}
}

func TestMiddlewareRejectsDifferentOwnerAndAllowsPublicPath(t *testing.T) {
	public := map[string]struct{}{"/healthz": {}}
	middleware := Middleware(fakeVerifier{principal: Principal{Subject: "other"}}, "owner-1", public)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "Bearer signed-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong owner returned %d", response.Code)
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("public route returned %d", response.Code)
	}
}
