package queue

import (
	"context"
	"errors"
	"sync"
)

type Message struct {
	Text string `json:"message"`
}

type Queue struct {
	mu          sync.Mutex
	items       []Message
	subscribers []chan Message
	maxMessages int
}

func New(max int) *Queue { return &Queue{maxMessages: max} }

func (q *Queue) Put(msg Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.subscribers) > 0 {
		ch := q.subscribers[0]
		q.subscribers = q.subscribers[1:]
		ch <- msg
		close(ch)
		return nil
	}

	if q.maxMessages > 0 && len(q.items) >= q.maxMessages {
		return errors.New("message limit exceeded")
	}
	q.items = append(q.items, msg)
	return nil
}

func (q *Queue) Get(ctx context.Context) (Message, error) {
	q.mu.Lock()
	if len(q.items) > 0 {
		msg := q.items[0]
		q.items = q.items[1:]
		q.mu.Unlock()
		return msg, nil
	}

	ch := make(chan Message, 1)
	q.subscribers = append(q.subscribers, ch)
	q.mu.Unlock()

	select {
	case msg := <-ch:
		return msg, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}
