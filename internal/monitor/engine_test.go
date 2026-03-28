package monitor

import (
	"context"
	"sync"
	"testing"
	"time"
)

type mockChecker struct {
	id     string
	status Status
	mu     sync.Mutex
	calls  int
}

func (m *mockChecker) ID() string { return m.id }

func (m *mockChecker) Check(ctx context.Context) CheckResult {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return CheckResult{
		MonitorID: m.id,
		Status:    m.status,
		Latency:   10 * time.Millisecond,
		Timestamp: time.Now(),
	}
}

func (m *mockChecker) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestEngine_RunAndStop(t *testing.T) {
	results := make(chan CheckResult, 100)
	engine := NewEngine(results)

	mock := &mockChecker{id: "mock-1", status: StatusHealthy}
	engine.Add(mock, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	engine.Start(ctx)

	// Wait for a few checks
	time.Sleep(200 * time.Millisecond)
	cancel()
	engine.Wait()

	if mock.CallCount() < 2 {
		t.Errorf("expected at least 2 calls, got %d", mock.CallCount())
	}

	// Drain results channel
	close(results)
	count := 0
	for range results {
		count++
	}
	if count < 2 {
		t.Errorf("expected at least 2 results, got %d", count)
	}
}

func TestEngine_Remove(t *testing.T) {
	results := make(chan CheckResult, 100)
	engine := NewEngine(results)

	mock := &mockChecker{id: "mock-1", status: StatusHealthy}
	engine.Add(mock, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	engine.Start(ctx)

	time.Sleep(100 * time.Millisecond)
	engine.Remove("mock-1")
	countAtRemove := mock.CallCount()

	time.Sleep(150 * time.Millisecond)
	cancel()
	engine.Wait()

	// Should not have received many more calls after removal
	if mock.CallCount() > countAtRemove+1 {
		t.Errorf("checker kept running after remove: %d calls (expected ~%d)", mock.CallCount(), countAtRemove)
	}
}
