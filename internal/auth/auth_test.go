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

type fakeBrowserSession struct{ principal Principal }

func (f fakeBrowserSession) Login(http.ResponseWriter, *http.Request)    {}
func (f fakeBrowserSession) Callback(http.ResponseWriter, *http.Request) {}
func (f fakeBrowserSession) Logout(http.ResponseWriter, *http.Request)   {}
func (f fakeBrowserSession) Principal(*http.Request) (Principal, bool) {
	return f.principal, f.principal.Subject != ""
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

func TestMiddlewareUsesBrowserSessionAndRedirectsPages(t *testing.T) {
	handler := Middleware(fakeVerifier{}, "owner-1", nil, fakeBrowserSession{principal: Principal{Subject: "owner-1"}})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/accounts", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("browser session returned %d", response.Code)
	}

	handler = Middleware(fakeVerifier{}, "owner-1", nil, fakeBrowserSession{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/accounts?sort=name", nil))
	if response.Code != http.StatusFound || response.Header().Get("Location") != "/auth/login?return_to=%2Faccounts%3Fsort%3Dname" {
		t.Fatalf("unexpected login redirect: %d %q", response.Code, response.Header().Get("Location"))
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("API request returned %d instead of 401", response.Code)
	}
}

func TestSafeReturnToRejectsExternalDestinations(t *testing.T) {
	for _, value := range []string{"", "https://attacker.example", "//attacker.example/path", "account"} {
		if got := safeReturnTo(value); got != "/" {
			t.Fatalf("safeReturnTo(%q) = %q", value, got)
		}
	}
	if got := safeReturnTo("/accounts?sort=name"); got != "/accounts?sort=name" {
		t.Fatalf("safe local return path became %q", got)
	}
}
