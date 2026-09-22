package broker

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestLoadFromDirMergesRegistryMetadata(t *testing.T) {
	dir := t.TempDir()
	base := "brokers:\n- id: example\n  name: Example\n  email: privacy@example.com\n  region: us\n"
	registry := "brokers:\n- id: example\n  name: Example Inc.\n  website: https://example.com\n  region: us\n  risk_flags: [biometric]\n  sources:\n  - registry: california-2026\n"
	if err := os.WriteFile(filepath.Join(dir, "brokers.yaml"), []byte(base), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(registry), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Brokers) != 1 {
		t.Fatalf("expected one merged broker, got %d", len(db.Brokers))
	}
	b := db.Brokers[0]
	if b.Email != "privacy@example.com" || b.Website != "https://example.com" {
		t.Fatalf("unexpected merge: %#v", b)
	}
	if len(b.Sources) != 1 || len(b.RiskFlags) != 1 {
		t.Fatalf("metadata not merged: %#v", b)
	}
}

func TestBundledCatalogLoadsAndHasUniqueIDs(t *testing.T) {
	db, err := LoadFromDir("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Brokers) < 1200 {
		t.Fatalf("expected at least 1,200 brokers, got %d", len(db.Brokers))
	}
	seen := map[string]bool{}
	registryBacked := 0
	for _, b := range db.Brokers {
		if b.ID == "" || b.Name == "" {
			t.Fatalf("broker missing identity: %#v", b)
		}
		if seen[b.ID] {
			t.Fatalf("duplicate merged broker ID %q", b.ID)
		}
		seen[b.ID] = true
		if len(b.Sources) > 0 {
			registryBacked++
		}
	}
	if registryBacked < 800 {
		t.Fatalf("expected registry provenance on at least 800 brokers, got %d", registryBacked)
	}
}
