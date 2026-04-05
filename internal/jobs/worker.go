package jobs

import (
	"context"
	"log/slog"
	"sync"
)

// Worker is a simple channel-based worker pool for background jobs.
type Worker struct {
	concurrency int
	queue       chan func(ctx context.Context)
	wg          sync.WaitGroup
}

func NewWorker(concurrency int) *Worker {
	return &Worker{
		concurrency: concurrency,
		queue:       make(chan func(ctx context.Context), 100),
	}
}

func (w *Worker) Start(ctx context.Context) {
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case fn, ok := <-w.queue:
					if !ok {
						return
					}
					fn(ctx)
				}
			}
		}()
	}
}

func (w *Worker) Enqueue(fn func(ctx context.Context)) {
	select {
	case w.queue <- fn:
	default:
		slog.Warn("worker queue full, dropping job")
	}
}

func (w *Worker) Stop() {
	close(w.queue)
	w.wg.Wait()
}
