package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eraser-privacy/eraser/internal/broker"
	"github.com/eraser-privacy/eraser/internal/config"
	"github.com/eraser-privacy/eraser/internal/history"
)

func TestExpandURL(t *testing.T) {
	got, err := expandURL("https://example.com/search?name={full_name}&city={city}", config.Profile{FirstName: "Jane", LastName: "Doe", City: "New York"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/search?name=Jane+Doe&city=New+York" {
		t.Fatalf("unexpected URL %q", got)
	}
}

func TestClassifyRequiresTwoSignalsForFound(t *testing.T) {
	p := config.Profile{FirstName: "Jane", LastName: "Doe", City: "Boston"}
	status, confidence, _ := classify("Profile for Jane Doe in Boston", p)
	if status != history.ExposureFound || confidence < 0.9 {
		t.Fatalf("got %s %.2f", status, confidence)
	}
	status, _, _ = classify("Profile for Jane Doe", p)
	if status != history.ExposurePossible {
		t.Fatalf("got %s", status)
	}
	status, _, _ = classify("No matching person", p)
	if status != history.ExposureNotFound {
		t.Fatalf("got %s", status)
	}
}

func TestWebChecker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "Jane+Doe") {
			t.Fatalf("profile was not expanded in query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte("Jane Doe lives in Boston"))
	}))
	defer server.Close()

	b := broker.Broker{ID: "example", Name: "Example", Workflow: broker.Workflow{Discovery: broker.WorkflowStep{Type: "web", URL: server.URL + "?q={full_name}"}}}
	result := DefaultRegistry(server.Client()).Check(context.Background(), b, config.Profile{FirstName: "Jane", LastName: "Doe", City: "Boston"})
	if result.Status != history.ExposureFound {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestUnsupportedDiscoveryFailsClosed(t *testing.T) {
	result := DefaultRegistry(nil).Check(context.Background(), broker.Broker{ID: "example"}, config.Profile{})
	if result.Status != history.ExposureUnknown || result.Error == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
