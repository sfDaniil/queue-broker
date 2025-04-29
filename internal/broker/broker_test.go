package broker_test

import (
	"context"
	"testing"
	"time"

	"queue-broker/internal/broker"
	"queue-broker/internal/queue"
)

// mustNoErr is a helper: fails t if err is non-nil.
func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// mustEqual fails if a != b.
func mustEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// TestPutCreatesQueue verifies that Put creates queue on-demand and stores the message.
func TestPutCreatesQueue(t *testing.T) {
	b := broker.New(0, 0)
	msg := queue.Message{Text: "hello"}
	mustNoErr(t, b.Put("q", msg))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := b.Get(ctx, "q")
	mustNoErr(t, err)
	mustEqual(t, got.Text, msg.Text)
}

// TestQueueLimit enforces maximum number of queues.
func TestQueueLimit(t *testing.T) {
	b := broker.New(1, 0)
	_ = b.Put("q1", queue.Message{Text: "a"})
	if err := b.Put("q2", queue.Message{Text: "b"}); err != broker.ErrQueueLimit {
		t.Fatalf("want ErrQueueLimit, got %v", err)
	}
}

// TestGetWaitsForQueue checks that Get blocks until the queue appears.
func TestGetWaitsForQueue(t *testing.T) {
	b := broker.New(0, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan queue.Message, 1)
	go func() {
		m, err := b.Get(ctx, "async")
		if err == nil {
			done <- m
		}
	}()

	time.Sleep(50 * time.Millisecond)
	_ = b.Put("async", queue.Message{Text: "delayed"})

	select {
	case m := <-done:
		mustEqual(t, m.Text, "delayed")
	case <-time.After(time.Second):
		t.Fatal("Get did not return after Put")
	}
}

// TestGetTimeoutWhenQueueMissing ensures deadline propagation when queue never appears.
func TestGetTimeoutWhenQueueMissing(t *testing.T) {
	b := broker.New(0, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := b.Get(ctx, "ghost")
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatalf("Get returned too early: %v", time.Since(start))
	}
}
