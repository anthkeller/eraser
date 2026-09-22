package history

import (
	"path/filepath"
	"testing"

	"github.com/eraser-privacy/eraser/internal/accounts"
)

func TestAccountInventoryUpsert(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	account := accounts.Account{Service: "Example", LoginURL: "https://example.com/login", Username: "jane", Source: "browser"}
	if err := store.UpsertAccount(account); err != nil {
		t.Fatal(err)
	}
	account.Service = "Example Updated"
	if err := store.UpsertAccount(account); err != nil {
		t.Fatal(err)
	}
	inventory, err := store.GetAccountInventory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory) != 1 || inventory[0].Service != "Example Updated" {
		t.Fatalf("unexpected inventory: %#v", inventory)
	}
}
