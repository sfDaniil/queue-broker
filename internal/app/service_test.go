package app_test

import (
	"context"
	"testing"

	"queue-broker/internal/adapters/memory"
	"queue-broker/internal/app"
)

func TestServicePutGet(t *testing.T) {
	repo := memory.NewRepo(0, 0)
	svc := app.NewService(repo)
	msg := app.Message{Text: "hello"}
	if err := svc.Put(context.Background(), "q", msg); err != nil {
		t.Fatalf("put error: %v", err)
	}
	got, err := svc.Get(context.Background(), "q")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if got != msg {
		t.Fatalf("expected %v, got %v", msg, got)
	}
}
