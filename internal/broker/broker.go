package broker

import (
	"context"
	"errors"
	"sync"
	"time"

	"queue-broker/internal/queue"
)

var (
	ErrQueueLimit = errors.New("queue limit exceeded")
)

type Broker struct {
	mu           sync.RWMutex
	queues       map[string]*queue.Queue
	maxQueues    int
	qMaxMessages int
}

func New(maxQ, maxPerQ int) *Broker {
	return &Broker{
		queues:       make(map[string]*queue.Queue),
		maxQueues:    maxQ,
		qMaxMessages: maxPerQ,
	}
}

func (b *Broker) getOrCreate(name string) (*queue.Queue, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if q, ok := b.queues[name]; ok {
		return q, nil
	}
	if b.maxQueues > 0 && len(b.queues) >= b.maxQueues {
		return nil, ErrQueueLimit
	}
	q := queue.New(b.qMaxMessages)
	b.queues[name] = q
	return q, nil
}

func (b *Broker) Put(name string, m queue.Message) error {
	q, err := b.getOrCreate(name)
	if err != nil {
		return err
	}
	return q.Put(m)
}

// Get: ждём появление очереди до deadline контекста
func (b *Broker) Get(ctx context.Context, name string) (queue.Message, error) {
	// быстрая попытка
	b.mu.RLock()
	q, ok := b.queues[name]
	b.mu.RUnlock()
	if ok {
		return q.Get(ctx)
	}

	// нет такой очереди ⇒ ждём её создания
	ready := make(chan struct{})
	go func() {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				b.mu.RLock()
				_, ok := b.queues[name]
				b.mu.RUnlock()
				if ok {
					close(ready)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case <-ready:
		// очередь появилась — пробуем снова
		return b.queues[name].Get(ctx)
	case <-ctx.Done():
		return queue.Message{}, ctx.Err() // DeadLineExceeded => вернём 404 выше
	}
}
