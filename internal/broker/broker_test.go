package broker

import "testing"

func TestRemovalMethodInference(t *testing.T) {
	if got := (Broker{Email: "privacy@example.com", OptOutURL: "https://example.com/remove"}).RemovalMethod(); got != RemovalEmail {
		t.Fatalf("got %q", got)
	}
	if got := (Broker{OptOutURL: "https://example.com/remove"}).RemovalMethod(); got != RemovalWebForm {
		t.Fatalf("got %q", got)
	}
	if got := (Broker{}).RemovalMethod(); got != RemovalManual {
		t.Fatalf("got %q", got)
	}
}

func TestExplicitRemovalMethod(t *testing.T) {
	b := Broker{Email: "privacy@example.com", Workflow: Workflow{Removal: WorkflowStep{Type: RemovalAPI, URL: "https://api.example.com/remove"}}}
	if got := b.RemovalMethod(); got != RemovalAPI {
		t.Fatalf("got %q", got)
	}
	if got := b.RemovalTarget(); got != "https://api.example.com/remove" {
		t.Fatalf("got %q", got)
	}
}
