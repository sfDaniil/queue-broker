package queue_test

import (
	"context"
	"queue-broker/internal/queue"
	"sync"
	"testing"
	"time"
)

// mustNoErr fails the test if err != nil
func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// mustEqual fails if a != b
func mustEqual[T comparable](t *testing.T, a, b T) {
	t.Helper()
	if a != b {
		t.Fatalf("want %v, got %v", b, a)
	}
}

// TestPutAndImmediateGet verifies that a message pushed before Get is returned without waiting.
func TestPutAndImmediateGet(t *testing.T) {
	q := &queue.Queue{}
	want := queue.Message{Text: "hello"}
	mustNoErr(t, q.Put(want))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := q.Get(ctx)
	mustNoErr(t, err)
	mustEqual(t, got.Text, want.Text)
}

// TestMessageLimit checks that Put enforces maxMessages.
func TestMessageLimit(t *testing.T) {
	queueTestSize := 1
	q := queue.New(queueTestSize)
	mustNoErr(t, q.Put(queue.Message{Text: "one"}))
	if err := q.Put(queue.Message{Text: "two"}); err == nil {
		t.Fatal("expected message limit error, got nil")
	}
}

// TestFIFO ensures FIFO order.
func TestFIFO(t *testing.T) {
	q := &queue.Queue{}
	msgs := []string{"first", "second", "third"}
	for _, m := range msgs {
		_ = q.Put(queue.Message{Text: m})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for _, want := range msgs {
		got, err := q.Get(ctx)
		mustNoErr(t, err)
		mustEqual(t, got.Text, want)
	}
}

// TestGetWaitUntilPut verifies that Get blocks until a message is added or context times out.
func TestGetWaitUntilPut(t *testing.T) {
	q := &queue.Queue{}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		_ = q.Put(queue.Message{Text: "async"})
	}()

	got, err := q.Get(ctx)
	mustNoErr(t, err)
	mustEqual(t, got.Text, "async")
	wg.Wait()
}

// TestGetTimeout ensures context deadline propagates to queue.
func TestGetTimeout(t *testing.T) {
	q := &queue.Queue{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := q.Get(ctx)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatalf("Get returned too early: %v", time.Since(start))
	}
}

// TestContextCancel ensures that cancellation unblocks Get.
func TestContextCancel(t *testing.T) {
	q := &queue.Queue{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = q.Get(ctx)
	}()
	time.Sleep(20 * time.Millisecond) // ensure goroutine is waiting
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Get did not unblock after context cancel")
	}
}

// TestConcurrentPutGet stresses thread‑safety.
func TestConcurrentPutGet(t *testing.T) {
	const msgCount = 1000
	const goCount = 10
	q := &queue.Queue{}
	var wg sync.WaitGroup

	// producers
	for i := 0; i < goCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < msgCount; j++ {
				_ = q.Put(queue.Message{Text: "m"})
			}
		}(i)
	}

	// consumers
	for i := 0; i < goCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < msgCount; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				_, err := q.Get(ctx)
				cancel()
				mustNoErr(t, err)
			}
		}()
	}
	wg.Wait()
}

// TestNoDuplicateAfterPut verifies that a single Put results in
// exactly one delivered message, even when a consumer is already
// waiting on Get.
func TestNoDuplicateAfterPut(t *testing.T) {
	q := &queue.Queue{}

	// 1 ) запустим первый Get, который должен получить сообщение.
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()

	got := make(chan queue.Message, 1)
	go func() {
		m, _ := q.Get(ctx1)
		got <- m
	}()

	// ждём, чтобы goroutine гарантированно заблокировалась внутри Get
	time.Sleep(10 * time.Millisecond)

	// 2 ) один-единственный Put
	wantMsg := queue.Message{Text: "unique"}
	mustNoErr(t, q.Put(wantMsg))

	// первый получатель обязан получить ровно это сообщение
	m := <-got
	mustEqual(t, m.Text, wantMsg.Text)

	// 3 ) запускаем второй Get с коротким тайм-аутом;
	// если бы сообщение осталось в буфере, он бы вернулся без ошибки.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	_, err := q.Get(ctx2)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded (queue empty), got %v", err)
	}
}
