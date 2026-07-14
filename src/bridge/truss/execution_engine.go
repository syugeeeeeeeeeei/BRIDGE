package truss

import (
	"context"
	"sync"
)

type TaskFunc func(context.Context) any
type ExecutionEngine struct{ Workers int }

func (e ExecutionEngine) Run(ctx context.Context, tasks []TaskFunc) []any {
	w := e.Workers
	if w < 1 {
		w = 1
	}
	out := make([]any, len(tasks))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < w; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					out[j] = tasks[j](ctx)
				}
			}
		}()
	}
	for i := range tasks {
		select {
		case jobs <- i:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return out
		}
	}
	close(jobs)
	wg.Wait()
	return out
}
