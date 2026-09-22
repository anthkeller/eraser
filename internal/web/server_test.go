package web

import (
	"path/filepath"
	"testing"

	"github.com/eraser-privacy/eraser/internal/broker"
	"github.com/eraser-privacy/eraser/internal/config"
	"github.com/eraser-privacy/eraser/internal/history"
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
