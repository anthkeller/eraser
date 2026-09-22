package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestExposurePersistenceAndDueChecks(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	check := &ExposureCheck{BrokerID: "future", BrokerName: "Future", Status: ExposureNotFound, Confidence: 0.8, CheckedAt: now, NextCheckAt: now.Add(24 * time.Hour)}
	if err := store.AddExposureCheck(check); err != nil {
		t.Fatal(err)
	}
	latest, err := store.GetLatestExposureCheck("future")
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || latest.Status != ExposureNotFound || latest.ID == 0 {
		t.Fatalf("unexpected latest check: %#v", latest)
	}

	due, err := store.GetDueBrokerIDs([]string{"future", "never"}, now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0] != "never" {
		t.Fatalf("unexpected due brokers: %#v", due)
	}
	checks, err := store.GetLatestExposureChecks(10)
	if err != nil || len(checks) != 1 || checks[0].BrokerID != "future" {
		t.Fatalf("unexpected checks: %#v err=%v", checks, err)
	}
}

func TestExposureTransitions(t *testing.T) {
	if got := TransitionExposureStatus(ExposureFound, ExposureNotFound); got != ExposureRemoved {
		t.Fatalf("expected removed, got %s", got)
	}
	if got := TransitionExposureStatus(ExposureNotFound, ExposureFound); got != ExposureReappeared {
		t.Fatalf("expected reappeared, got %s", got)
	}
	if got := TransitionExposureStatus(ExposureFound, ExposureFound); got != ExposureFound {
		t.Fatalf("expected unchanged found, got %s", got)
	}
}
