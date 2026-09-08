package concurrency

import (
	"context"
	"sync"
	"testing"
)

func TestWorkerPoolProcessesJobsConcurrently(t *testing.T) {
	ctx := context.Background()
	var mu sync.Mutex
	processed := make(map[int]bool)

	pool := NewWorkerPool[int](4, 8, func(ctx context.Context, job int) error {
		mu.Lock()
		processed[job] = true
		mu.Unlock()
		return nil
	})

	errors := pool.Start(ctx)
	for i := 0; i < 20; i++ {
		if err := pool.Submit(ctx, i); err != nil {
			t.Fatal(err)
		}
	}

	pool.Close()
	for range errors {
	}

	if len(processed) != 20 {
		t.Fatalf("expected 20 processed jobs, got %d", len(processed))
	}
}
