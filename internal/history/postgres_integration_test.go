package history

import (
	"os"
	"testing"
	"time"
)

func TestPostgresRowLevelOwnerIsolation(t *testing.T) {
	dsn := os.Getenv("ERASER_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("ERASER_TEST_POSTGRES_URL is not configured")
	}
	ownerA, err := NewPostgresStore(dsn, "test-owner-a")
	if err != nil {
		t.Fatal(err)
	}
	defer ownerA.Close()
	ownerB, err := NewPostgresStore(dsn, "test-owner-b")
	if err != nil {
		t.Fatal(err)
	}
	defer ownerB.Close()

	recordA := &Record{BrokerID: "owner-a-broker", BrokerName: "Owner A", Email: "privacy@example.com", Template: "generic", Status: StatusSent, SentAt: time.Now().UTC()}
	if err := ownerA.Add(recordA); err != nil {
		t.Fatal(err)
	}
	visibleToB, err := ownerB.GetRecentRequests(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleToB) != 0 {
		t.Fatalf("owner B could read %d owner A records", len(visibleToB))
	}

	recordB := &Record{BrokerID: "owner-b-broker", BrokerName: "Owner B", Email: "privacy@example.com", Template: "generic", Status: StatusSent, SentAt: time.Now().UTC()}
	if err := ownerB.Add(recordB); err != nil {
		t.Fatal(err)
	}
	visibleToA, err := ownerA.GetRecentRequests(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleToA) != 1 || visibleToA[0].BrokerID != "owner-a-broker" {
		t.Fatalf("owner A isolation failed: %+v", visibleToA)
	}

	deleted, err := ownerB.DeleteByStatus(StatusSent)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("owner B deleted %d records, expected only its own", deleted)
	}
	visibleToA, err = ownerA.GetRecentRequests(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleToA) != 1 {
		t.Fatal("owner B deleted owner A data")
	}
}
