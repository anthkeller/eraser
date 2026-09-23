package work

import (
	"context"
	"testing"
)

func TestMemoryQueuePreservesOwnerAndDispatches(t *testing.T) {
	queue := NewMemoryQueue(1)
	defer queue.Close()
	job, err := queue.Enqueue(context.Background(), Job{OwnerID: "owner-1", Type: TypeExposureScan})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" || job.CreatedAt.IsZero() {
		t.Fatal("queue did not assign job metadata")
	}
	called := false
	worker := NewWorker(queue, map[Type]Handler{
		TypeExposureScan: func(_ context.Context, received Job) error {
			called = received.OwnerID == "owner-1"
			return nil
		},
	})
	if err := worker.RunOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("owner-scoped job was not dispatched")
	}
}

func TestMemoryQueueRejectsOwnerlessJob(t *testing.T) {
	queue := NewMemoryQueue(1)
	defer queue.Close()
	if _, err := queue.Enqueue(context.Background(), Job{Type: TypeExposureScan}); err == nil {
		t.Fatal("expected ownerless job to be rejected")
	}
}
