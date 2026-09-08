package concurrency

import (
	"context"
	"sync"
)

type WorkerPool[T any] struct {
	jobs    chan T
	handler func(context.Context, T) error
	workers int
	wg      sync.WaitGroup
}

func NewWorkerPool[T any](workerCount int, buffer int, handler func(context.Context, T) error) *WorkerPool[T] {
	if workerCount < 1 {
		workerCount = 1
	}

	pool := &WorkerPool[T]{
		jobs:    make(chan T, buffer),
		handler: handler,
		workers: workerCount,
	}

	return pool
}

func (p *WorkerPool[T]) Start(ctx context.Context) <-chan error {
	errors := make(chan error, p.workers)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					if err := p.handler(ctx, job); err != nil {
						select {
						case errors <- err:
						case <-ctx.Done():
							return
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		p.wg.Wait()
		close(errors)
	}()

	return errors
}

func (p *WorkerPool[T]) Submit(ctx context.Context, job T) error {
	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *WorkerPool[T]) Close() {
	close(p.jobs)
	p.wg.Wait()
}
