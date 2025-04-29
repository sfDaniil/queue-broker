package app

import "context"

type Service struct {
	repo QueueRepository
}

func NewService(repo QueueRepository) *Service { return &Service{repo: repo} }

func (s *Service) Put(ctx context.Context, queue string, msg Message) error {
	return s.repo.Enqueue(ctx, queue, msg)
}

func (s *Service) Get(ctx context.Context, queue string) (Message, error) {
	return s.repo.Dequeue(ctx, queue)
}
