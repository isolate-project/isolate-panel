package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Pool is a bounded worker pool for concurrent task execution.
type Pool struct {
	maxWorkers int
	sem        chan struct{}
	mu         sync.Mutex
	err        error
}

// NewPool creates a worker pool with the given max concurrency.
// If maxWorkers <= 0, defaults to runtime.GOMAXPROCS(0).
func NewPool(maxWorkers int) *Pool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.GOMAXPROCS(0)
	}
	return &Pool{
		maxWorkers: maxWorkers,
		sem:        make(chan struct{}, maxWorkers),
	}
}

// Go submits a task to the pool. It blocks if all workers are busy.
// The first error from any task is retained and subsequent tasks are skipped.
func (p *Pool) Go(ctx context.Context, fn func() error) error {
	p.mu.Lock()
	if p.err != nil {
		p.mu.Unlock()
		return p.err
	}
	p.mu.Unlock()

	select {
	case p.sem <- struct{}{}:
		go func() {
			defer func() { <-p.sem }()
			if err := fn(); err != nil {
				p.mu.Lock()
				if p.err == nil {
					p.err = err
				}
				p.mu.Unlock()
			}
		}()
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Wait blocks until all submitted tasks complete.
func (p *Pool) Wait() error {
	for i := 0; i < p.maxWorkers; i++ {
		p.sem <- struct{}{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

// ErrgroupPool wraps errgroup.Group with bounded concurrency.
type ErrgroupPool struct {
	g   *errgroup.Group
	sem chan struct{}
	ctx context.Context
}

// NewErrgroupPool creates an errgroup with limited concurrency.
func NewErrgroupPool(ctx context.Context, maxWorkers int) *ErrgroupPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.GOMAXPROCS(0)
	}
	g, ctx := errgroup.WithContext(ctx)
	return &ErrgroupPool{
		g:   g,
		sem: make(chan struct{}, maxWorkers),
		ctx: ctx,
	}
}

// Go submits a task. Blocks if concurrency limit reached.
func (p *ErrgroupPool) Go(fn func() error) {
	p.g.Go(func() error {
		p.sem <- struct{}{}
		defer func() { <-p.sem }()
		return fn()
	})
}

// Wait blocks until all tasks complete.
func (p *ErrgroupPool) Wait() error {
	return p.g.Wait()
}

// Context returns the pool's context (cancels on first error).
func (p *ErrgroupPool) Context() context.Context {
	return p.ctx
}

// ResultCollector runs tasks and collects their results.
type ResultCollector[T any] struct {
	mu      sync.Mutex
	results []T
	err     error
}

// NewResultCollector creates a collector for type T.
func NewResultCollector[T any]() *ResultCollector[T] {
	return &ResultCollector[T]{}
}

// Add stores a result or error.
func (rc *ResultCollector[T]) Add(result T, err error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if err != nil {
		if rc.err == nil {
			rc.err = err
		}
		return
	}
	rc.results = append(rc.results, result)
}

// Results returns all collected results.
func (rc *ResultCollector[T]) Results() []T {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	out := make([]T, len(rc.results))
	copy(out, rc.results)
	return out
}

// Error returns the first error encountered.
func (rc *ResultCollector[T]) Error() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.err
}

// SafeRun executes fn, recovering from panics and returning them as errors.
func SafeRun(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			err = fmt.Errorf("panic: %v\n%s", r, buf[:n])
		}
	}()
	return fn()
}
