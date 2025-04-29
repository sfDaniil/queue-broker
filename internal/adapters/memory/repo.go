package memory

import (
	"context"
	"errors"
	"sync"

	"queue-broker/internal/app"
)

type queueItem struct {
	messages []app.Message
	subs     []chan app.Message
}

type Repo struct {
	mu          sync.RWMutex
	queues      map[string]*queueItem
	maxQueues   int
	maxMessages int
	active      int // count of active queues
}

func NewRepo(maxQ, maxPerQ int) *Repo {
	return &Repo{
		queues:      make(map[string]*queueItem),
		maxQueues:   maxQ,
		maxMessages: maxPerQ,
	}
}

func (r *Repo) cleanup(name string, queue *queueItem) {
	if len(queue.messages) == 0 && len(queue.subs) == 0 {
		delete(r.queues, name)
		r.active--
	}
}

func (r *Repo) getOrCreate(name string) *queueItem {
	if queue, ok := r.queues[name]; ok {
		return queue
	}
	queue := &queueItem{}
	r.queues[name] = queue
	return queue
}

func (r *Repo) Enqueue(ctx context.Context, name string, msg app.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	queue := r.getOrCreate(name)

	if len(queue.messages) == 0 && len(queue.subs) == 0 {
		if r.maxQueues > 0 && r.active >= r.maxQueues {
			return errors.New("queue limit exceeded")
		}
		r.active++
	}

	if len(queue.subs) > 0 {
		ch := queue.subs[0]
		queue.subs = queue.subs[1:]
		ch <- msg
		close(ch)
		r.cleanup(name, queue)
		return nil
	}

	if r.maxMessages > 0 && len(queue.messages) >= r.maxMessages {
		return errors.New("message limit exceeded")
	}

	queue.messages = append(queue.messages, msg)
	return nil
}

func (r *Repo) Dequeue(ctx context.Context, name string) (app.Message, error) {
	r.mu.Lock()
	queue, ok := r.queues[name]
	if ok && len(queue.messages) > 0 {
		msg := queue.messages[0]
		queue.messages = queue.messages[1:]
		r.cleanup(name, queue)
		r.mu.Unlock()
		return msg, nil
	}

	// subscribe
	ch := make(chan app.Message, 1)
	if !ok {
		queue = r.getOrCreate(name)
	}
	queue.subs = append(queue.subs, ch)
	if len(queue.subs) == 1 && len(queue.messages) == 0 {
		if r.maxQueues > 0 && r.active >= r.maxQueues {
			queue.subs = queue.subs[:len(queue.subs)-1]
			r.mu.Unlock()
			return app.Message{}, errors.New("queue limit exceeded")
		}
		r.active++
	}
	r.mu.Unlock()

	select {
	case msg := <-ch:
		return msg, nil
	case <-ctx.Done():
		// unsubscribe
		r.mu.Lock()
		if qItem2, ok2 := r.queues[name]; ok2 {
			for i, c := range qItem2.subs {
				if c == ch {
					qItem2.subs = append(qItem2.subs[:i], qItem2.subs[i+1:]...)
					break
				}
			}
			r.cleanup(name, qItem2)
		}
		r.mu.Unlock()
		return app.Message{}, ctx.Err()
	}
}
