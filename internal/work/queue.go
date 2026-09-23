package work

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeExposureScan     Type = "exposure.scan"
	TypeBrokerValidation Type = "broker.validate"
	TypeRemovalSend      Type = "removal.send"
	TypeRecurringCheck   Type = "exposure.recheck"
)

var ErrClosed = errors.New("work queue is closed")

type Job struct {
	ID        string          `json:"id"`
	OwnerID   string          `json:"owner_id"`
	Type      Type            `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type Delivery struct {
	Job Job
	Ack func(error) error
}

type Queue interface {
	Enqueue(context.Context, Job) (Job, error)
	Receive(context.Context) (Delivery, error)
	Close() error
}

type MemoryQueue struct {
	jobs      chan Job
	closed    chan struct{}
	closeOnce sync.Once
}

func NewMemoryQueue(capacity int) *MemoryQueue {
	if capacity < 1 {
		capacity = 1
	}
	return &MemoryQueue{jobs: make(chan Job, capacity), closed: make(chan struct{})}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job Job) (Job, error) {
	if strings.TrimSpace(job.OwnerID) == "" {
		return Job{}, errors.New("job owner ID is required")
	}
	if strings.TrimSpace(string(job.Type)) == "" {
		return Job{}, errors.New("job type is required")
	}
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	select {
	case <-ctx.Done():
		return Job{}, ctx.Err()
	case <-q.closed:
		return Job{}, ErrClosed
	case q.jobs <- job:
		return job, nil
	}
}

func (q *MemoryQueue) Receive(ctx context.Context) (Delivery, error) {
	select {
	case <-ctx.Done():
		return Delivery{}, ctx.Err()
	case <-q.closed:
		return Delivery{}, ErrClosed
	case job := <-q.jobs:
		return Delivery{Job: job, Ack: func(error) error { return nil }}, nil
	}
}

func (q *MemoryQueue) Close() error {
	q.closeOnce.Do(func() { close(q.closed) })
	return nil
}

type Handler func(context.Context, Job) error

type Worker struct {
	queue    Queue
	handlers map[Type]Handler
}

func NewWorker(queue Queue, handlers map[Type]Handler) *Worker {
	return &Worker{queue: queue, handlers: handlers}
}

func (w *Worker) RunOne(ctx context.Context) error {
	delivery, err := w.queue.Receive(ctx)
	if err != nil {
		return err
	}
	handler, ok := w.handlers[delivery.Job.Type]
	if !ok {
		err = fmt.Errorf("no handler registered for job type %q", delivery.Job.Type)
	} else {
		err = handler(ctx, delivery.Job)
	}
	if ackErr := delivery.Ack(err); ackErr != nil {
		return fmt.Errorf("acknowledge job: %w", ackErr)
	}
	return err
}
