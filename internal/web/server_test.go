package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/eraser-privacy/eraser/internal/auth"
	"github.com/eraser-privacy/eraser/internal/broker"
	"github.com/eraser-privacy/eraser/internal/config"
	"github.com/eraser-privacy/eraser/internal/history"
	"github.com/eraser-privacy/eraser/internal/product"
	emaTemplate "github.com/eraser-privacy/eraser/internal/template"
)

type fixedVerifier struct{ subject string }

func (v fixedVerifier) Verify(_ context.Context, _ string) (auth.Principal, error) {
	return auth.Principal{Subject: v.subject}, nil
}

func TestNewServerParsesAllTemplates(t *testing.T) {
	store, err := history.NewStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	engine, err := emaTemplate.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(8080, &config.Config{}, "", &broker.BrokerDatabase{}, store, engine)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"intelligence.html", "accounts.html"} {
		if server.templates[name] == nil {
			t.Fatalf("template %s was not parsed", name)
		}
	}
}

func TestOperationalAndSystemEndpoints(t *testing.T) {
	store, err := history.NewStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	engine, err := emaTemplate.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	settings := product.Settings{
		Edition: product.EditionCommercial, Deployment: product.DeploymentSelfHosted,
		BindAddress: "127.0.0.1", OwnerID: "secret-owner", OpenBrowser: false,
	}
	server, err := NewServer(8080, &config.Config{}, "", &broker.BrokerDatabase{}, store, engine, WithProductSettings(settings))
	if err != nil {
		t.Fatal(err)
	}
	router := server.setupRouter()
	for _, path := range []string{"/healthz", "/readyz"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, response.Code)
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/system", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("system endpoint returned %d", response.Code)
	}
	var info product.PublicInfo
	if err := json.NewDecoder(response.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Edition != product.EditionCommercial || !info.Capabilities.SingleUser {
		t.Fatalf("unexpected product information: %+v", info)
	}
}

func TestHostedServerRequiresVerifiedConfiguredOwner(t *testing.T) {
	store, err := history.NewStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	engine, err := emaTemplate.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	settings := product.Settings{
		Edition: product.EditionCloud, Deployment: product.DeploymentHosted,
		BindAddress: "127.0.0.1", PublicURL: "https://eraser.example",
		OwnerID: "owner-1", AuthIssuer: "https://id.example", AuthAudience: "eraser",
	}
	server, err := NewServer(8080, &config.Config{}, "", &broker.BrokerDatabase{}, store, engine,
		WithProductSettings(settings), WithAuthVerifier(fixedVerifier{subject: "owner-1"}))
	if err != nil {
		t.Fatal(err)
	}
	router := server.setupRouter()

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request returned %d", response.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authenticated request returned %d", response.Code)
	}
	var principal auth.Principal
	if err := json.NewDecoder(response.Body).Decode(&principal); err != nil {
		t.Fatal(err)
	}
	if principal.Subject != "owner-1" {
		t.Fatalf("unexpected subject %q", principal.Subject)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("public health endpoint returned %d", response.Code)
	}
}
