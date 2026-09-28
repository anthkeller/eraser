package history

import (
	"path/filepath"
	"testing"
)

func TestPostgresPlaceholderBinding(t *testing.T) {
	db := &database{driver: "postgres"}
	query := db.bind("SELECT * FROM records WHERE owner = ? AND status = ? LIMIT ?")
	if query != "SELECT * FROM records WHERE owner = $1 AND status = $2 LIMIT $3" {
		t.Fatalf("unexpected bound query: %s", query)
	}
	if got := (&database{driver: "sqlite"}).bind("SELECT ?"); got != "SELECT ?" {
		t.Fatalf("SQLite query changed: %s", got)
	}
}

func TestSQLiteMigrationAddsOwnershipColumns(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, table := range []string{"removal_requests", "broker_responses", "pending_tasks", "exposure_checks", "broker_validations", "account_inventory"} {
		rows, err := store.db.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for rows.Next() {
			var cid int
			var name, columnType string
			var notNull, primaryKey int
			var defaultValue any
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			if name == "owner_id" {
				found = true
			}
		}
		rows.Close()
		if !found {
			t.Fatalf("%s has no owner_id column", table)
		}
	}
}
