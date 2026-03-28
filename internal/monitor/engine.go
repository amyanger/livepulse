package monitor

import (
	"context"
	"sync"
	"time"
)

type monitorEntry struct {
	checker  Checker
	interval time.Duration
	cancel   context.CancelFunc
}

type Engine struct {
	results  chan<- CheckResult
	monitors map[string]*monitorEntry
	mu       sync.Mutex
	wg       sync.WaitGroup
}

func NewEngine(results chan<- CheckResult) *Engine {
	return &Engine{
		results:  results,
		monitors: make(map[string]*monitorEntry),
	}
}

func (e *Engine) Add(checker Checker, interval time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Remove existing if present
	if existing, ok := e.monitors[checker.ID()]; ok {
		existing.cancel()
	}

	e.monitors[checker.ID()] = &monitorEntry{
		checker:  checker,
		interval: interval,
	}
}

func (e *Engine) Remove(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if entry, ok := e.monitors[id]; ok {
		if entry.cancel != nil {
			entry.cancel()
		}
		delete(e.monitors, id)
	}
}

func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, entry := range e.monitors {
		e.startMonitor(ctx, entry)
	}
}

func (e *Engine) startMonitor(ctx context.Context, entry *monitorEntry) {
	monCtx, cancel := context.WithCancel(ctx)
	entry.cancel = cancel

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		ticker := time.NewTicker(entry.interval)
		defer ticker.Stop()

		// Run immediately on start
		result := entry.checker.Check(monCtx)
		select {
		case e.results <- result:
		case <-monCtx.Done():
			return
		}

		for {
			select {
			case <-monCtx.Done():
				return
			case <-ticker.C:
				result := entry.checker.Check(monCtx)
				select {
				case e.results <- result:
				case <-monCtx.Done():
					return
				}
			}
		}
	}()
}

func (e *Engine) Wait() {
	e.wg.Wait()
}
