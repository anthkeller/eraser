package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/eraser-privacy/eraser/internal/broker"
	"github.com/eraser-privacy/eraser/internal/config"
	"github.com/eraser-privacy/eraser/internal/history"
	"github.com/eraser-privacy/eraser/internal/product"
	emaTemplate "github.com/eraser-privacy/eraser/internal/template"
)

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
