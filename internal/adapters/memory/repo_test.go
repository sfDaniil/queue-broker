package memory

import (
	"context"
	"testing"
	"time"

	"queue-broker/internal/app"
)

func TestEnqueueDequeue(t *testing.T) {
	repo := NewRepo(0, 0)
	msg := app.Message{Text: "hello"}
	if err := repo.Enqueue(context.Background(), "q1", msg); err != nil {
		t.Fatalf("enqueue error: %v", err)
	}
	got, err := repo.Dequeue(context.Background(), "q1")
	if err != nil {
		t.Fatalf("dequeue error: %v", err)
	}
	if got != msg {
		t.Fatalf("expected %v, got %v", msg, got)
	}
}

func TestMessageLimit(t *testing.T) {
	repo := NewRepo(0, 1)
	ctx := context.Background()
	if err := repo.Enqueue(ctx, "q1", app.Message{Text: "m1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Enqueue(ctx, "q1", app.Message{Text: "m2"}); err == nil {
		t.Fatalf("expected message limit error, got nil")
	}
}

func TestQueueLimit(t *testing.T) {
	repo := NewRepo(1, 0)
	ctx := context.Background()
	_ = repo.Enqueue(ctx, "q1", app.Message{Text: "a"})
	if err := repo.Enqueue(ctx, "q2", app.Message{Text: "b"}); err == nil {
		t.Fatalf("expected queue limit error")
	}
}

func TestSubscribeDelivery(t *testing.T) {
	repo := NewRepo(0, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		msg, err := repo.Dequeue(ctx, "q1")
		if err != nil {
			t.Errorf("dequeue error: %v", err)
		}
		if msg.Text != "hello" {
			t.Errorf("expected hello, got %s", msg.Text)
		}
		close(done)
	}()
	// ensure goroutine is waiting
	time.Sleep(20 * time.Millisecond)
	if err := repo.Enqueue(context.Background(), "q1", app.Message{Text: "hello"}); err != nil {
		t.Fatalf("enqueue error: %v", err)
	}
	<-done
}
