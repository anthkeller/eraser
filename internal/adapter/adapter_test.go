package adapter

import (
	"testing"

	"github.com/eraser-privacy/eraser/internal/broker"
)

func TestDefaultRegistryLegacyCompatibility(t *testing.T) {
	tests := []struct {
		name, email, url, method, target string
	}{
		{"email wins", "privacy@example.com", "https://example.com/optout", broker.RemovalEmail, "privacy@example.com"},
		{"form fallback", "", "https://example.com/optout", broker.RemovalWebForm, "https://example.com/optout"},
		{"manual fallback", "", "", broker.RemovalManual, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := DefaultRegistry().Plan(broker.Broker{ID: "example", Email: tt.email, OptOutURL: tt.url})
			if err != nil {
				t.Fatal(err)
			}
			if action.Method != tt.method || action.Target != tt.target {
				t.Fatalf("got method=%q target=%q", action.Method, action.Target)
			}
		})
	}
}

func TestExplicitWorkflowOverridesLegacyFields(t *testing.T) {
	b := broker.Broker{ID: "example", Email: "privacy@example.com", Workflow: broker.Workflow{Removal: broker.WorkflowStep{Type: broker.RemovalWebForm, URL: "https://example.com/remove"}}}
	action, err := DefaultRegistry().Plan(b)
	if err != nil {
		t.Fatal(err)
	}
	if action.Method != broker.RemovalWebForm || action.Target != "https://example.com/remove" {
		t.Fatalf("unexpected action: %#v", action)
	}
}

func TestUnknownAdapterFailsClosed(t *testing.T) {
	b := broker.Broker{ID: "example", Email: "privacy@example.com", Workflow: broker.Workflow{Removal: broker.WorkflowStep{Type: broker.RemovalEmail, Adapter: "custom-missing"}}}
	if _, err := DefaultRegistry().Plan(b); err == nil {
		t.Fatal("expected an error")
	}
}
