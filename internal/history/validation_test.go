package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBrokerValidationPersistence(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first := &BrokerValidation{BrokerID: "example", WebsiteValid: false, CheckedAt: time.Now().Add(-time.Hour)}
	second := &BrokerValidation{BrokerID: "example", WebsiteValid: true, OptOutValid: true, EmailDomainValid: true, CheckedAt: time.Now()}
	if err := store.AddBrokerValidation(first); err != nil {
		t.Fatal(err)
	}
	if err := store.AddBrokerValidation(second); err != nil {
		t.Fatal(err)
	}
	latest, err := store.GetLatestBrokerValidations()
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 1 || !latest["example"].WebsiteValid {
		t.Fatalf("unexpected latest validation: %#v", latest)
	}
}
