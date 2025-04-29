package app

import "context"

type Message struct {
	Text string `json:"message"`
}

type Producer interface {
	Put(ctx context.Context, queue string, msg Message) error
}

type Consumer interface {
	Get(ctx context.Context, queue string) (Message, error)
}

type QueueRepository interface {
	Enqueue(ctx context.Context, queue string, msg Message) error
	Dequeue(ctx context.Context, queue string) (Message, error)
}
