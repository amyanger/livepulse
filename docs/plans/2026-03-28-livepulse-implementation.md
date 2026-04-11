# livepulse Implementation Plan


**Goal:** Build a real-time infrastructure monitor with a full interactive TUI dashboard as a single Go binary.

**Architecture:** Elm-architecture TUI (bubbletea v2) with concurrent monitor goroutines feeding results through channels to an in-memory state store. Monitors persist to `~/.livepulse/monitors.json`. Alert state machine evaluates transitions and dispatches visual/desktop/webhook alerts.

**Tech Stack:** Go 1.26, bubbletea v2, lipgloss v2, bubbles v2, beeep, gopsutil

**API Notes (v2 specifics):**
- bubbletea v2: `tea.KeyPressMsg` (not `tea.KeyMsg`), `msg.String()` for key matching
- lipgloss v2: import `charm.land/lipgloss/v2`
- bubbles v2: `SetWidth()`/`SetHeight()` methods (not direct fields), `textinput.DefaultKeyMap()` is a function call

---

### Task 1: Store — Persistence Layer

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

**Step 1: Write the failing test**

```go
// internal/store/store_test.go
package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "monitors.json")

	s := New(path)

	m := MonitorConfig{
		ID:       "test-1",
		Name:     "Test HTTP",
		Type:     "http",
		Target:   "https://example.com",
		Interval: 10,
		Alerts: AlertConfig{
			Visual:  true,
			Desktop: false,
			Webhook: "",
		},
	}

	if err := s.Save([]MonitorConfig{m}); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(loaded))
	}
	if loaded[0].ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", loaded[0].ID)
	}
	if loaded[0].Target != "https://example.com" {
		t.Errorf("expected target https://example.com, got %s", loaded[0].Target)
	}
}

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "monitors.json")

	s := New(path)
	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected 0 monitors, got %d", len(loaded))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -v`
Expected: FAIL — types and functions not defined

**Step 3: Write minimal implementation**

```go
// internal/store/store.go
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type AlertConfig struct {
	Visual  bool   `json:"visual"`
	Desktop bool   `json:"desktop"`
	Webhook string `json:"webhook,omitempty"`
}

type MonitorConfig struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Target   string      `json:"target"`
	Interval int         `json:"interval"`
	Alerts   AlertConfig `json:"alerts"`
	// Custom check fields
	Command        string `json:"command,omitempty"`
	ExpectedStatus int    `json:"expected_status,omitempty"`
}

type Store struct {
	path string
}

func New(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Save(monitors []MonitorConfig) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}

	data, err := json.MarshalIndent(monitors, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal monitors: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write monitors: %w", err)
	}
	return nil
}

func (s *Store) Load() ([]MonitorConfig, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []MonitorConfig{}, nil
		}
		return nil, fmt.Errorf("read monitors: %w", err)
	}

	var monitors []MonitorConfig
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, fmt.Errorf("unmarshal monitors: %w", err)
	}
	return monitors, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat: add persistence store for monitor configs"
```

---

### Task 2: Check Interface + HTTP Check

**Files:**
- Create: `internal/monitor/check.go`
- Create: `internal/monitor/http.go`
- Test: `internal/monitor/http_test.go`

**Step 1: Write check interface and result types**

```go
// internal/monitor/check.go
package monitor

import (
	"context"
	"time"
)

type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
)

type CheckResult struct {
	MonitorID string
	Status    Status
	Latency   time.Duration
	Error     string
	Timestamp time.Time
}

type Checker interface {
	Check(ctx context.Context) CheckResult
	ID() string
}
```

**Step 2: Write the failing HTTP check test**

```go
// internal/monitor/http_test.go
package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewHTTPCheck("test-http", srv.URL, http.StatusOK, 5*time.Second)
	result := c.Check(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s", result.Status)
	}
	if result.Latency <= 0 {
		t.Error("expected positive latency")
	}
	if result.MonitorID != "test-http" {
		t.Errorf("expected monitor ID test-http, got %s", result.MonitorID)
	}
}

func TestHTTPCheck_Down(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewHTTPCheck("test-http", srv.URL, http.StatusOK, 5*time.Second)
	result := c.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected down, got %s", result.Status)
	}
}

func TestHTTPCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewHTTPCheck("test-http", srv.URL, http.StatusOK, 100*time.Millisecond)
	result := c.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected down on timeout, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message on timeout")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/monitor/ -v -run TestHTTP`
Expected: FAIL — `NewHTTPCheck` not defined

**Step 4: Write HTTP check implementation**

```go
// internal/monitor/http.go
package monitor

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type HTTPCheck struct {
	id             string
	url            string
	expectedStatus int
	timeout        time.Duration
	client         *http.Client
}

func NewHTTPCheck(id, url string, expectedStatus int, timeout time.Duration) *HTTPCheck {
	return &HTTPCheck{
		id:             id,
		url:            url,
		expectedStatus: expectedStatus,
		timeout:        timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HTTPCheck) ID() string { return c.id }

func (c *HTTPCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("create request: %v", err)
		result.Latency = time.Since(start)
		return result
	}

	resp, err := c.client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != c.expectedStatus {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("expected status %d, got %d", c.expectedStatus, resp.StatusCode)
		return result
	}

	result.Status = StatusHealthy
	return result
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/monitor/ -v -run TestHTTP`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/monitor/check.go internal/monitor/http.go internal/monitor/http_test.go
git commit -m "feat: add check interface and HTTP check implementation"
```

---

### Task 3: TCP Check

**Files:**
- Create: `internal/monitor/tcp.go`
- Test: `internal/monitor/tcp_test.go`

**Step 1: Write the failing test**

```go
// internal/monitor/tcp_test.go
package monitor

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPCheck_Healthy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	c := NewTCPCheck("test-tcp", ln.Addr().String(), 2*time.Second)
	result := c.Check(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s (err: %s)", result.Status, result.Error)
	}
}

func TestTCPCheck_Down(t *testing.T) {
	c := NewTCPCheck("test-tcp", "127.0.0.1:1", 500*time.Millisecond)
	result := c.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected down, got %s", result.Status)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/monitor/ -v -run TestTCP`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/monitor/tcp.go
package monitor

import (
	"context"
	"fmt"
	"net"
	"time"
)

type TCPCheck struct {
	id      string
	address string
	timeout time.Duration
}

func NewTCPCheck(id, address string, timeout time.Duration) *TCPCheck {
	return &TCPCheck{id: id, address: address, timeout: timeout}
}

func (c *TCPCheck) ID() string { return c.id }

func (c *TCPCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.address)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("tcp connect: %v", err)
		return result
	}
	conn.Close()

	result.Status = StatusHealthy
	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/monitor/ -v -run TestTCP`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/monitor/tcp.go internal/monitor/tcp_test.go
git commit -m "feat: add TCP port check"
```

---

### Task 4: System Metrics Check

**Files:**
- Create: `internal/monitor/system.go`
- Test: `internal/monitor/system_test.go`

**Step 1: Install gopsutil dependency**

Run: `go get github.com/shirou/gopsutil/v4/cpu github.com/shirou/gopsutil/v4/mem github.com/shirou/gopsutil/v4/disk`

**Step 2: Write the failing test**

```go
// internal/monitor/system_test.go
package monitor

import (
	"context"
	"testing"
)

func TestSystemCheck_ReturnsMetrics(t *testing.T) {
	c := NewSystemCheck("test-sys")
	result := c.Check(context.Background())

	if result.Status != StatusHealthy && result.Status != StatusDegraded {
		t.Errorf("expected healthy or degraded, got %s (err: %s)", result.Status, result.Error)
	}
	if result.MonitorID != "test-sys" {
		t.Errorf("expected monitor ID test-sys, got %s", result.MonitorID)
	}
}

func TestSystemCheck_Metrics(t *testing.T) {
	c := NewSystemCheck("test-sys")
	metrics, err := c.Collect()
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if metrics.CPUPercent < 0 || metrics.CPUPercent > 100 {
		t.Errorf("cpu percent out of range: %f", metrics.CPUPercent)
	}
	if metrics.MemPercent < 0 || metrics.MemPercent > 100 {
		t.Errorf("mem percent out of range: %f", metrics.MemPercent)
	}
	if metrics.DiskPercent < 0 || metrics.DiskPercent > 100 {
		t.Errorf("disk percent out of range: %f", metrics.DiskPercent)
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/monitor/ -v -run TestSystem`
Expected: FAIL

**Step 4: Write implementation**

```go
// internal/monitor/system.go
package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemMetrics struct {
	CPUPercent  float64
	MemPercent  float64
	MemUsedGB   float64
	MemTotalGB  float64
	DiskPercent float64
	DiskUsedGB  float64
	DiskTotalGB float64
}

type SystemCheck struct {
	id string
}

func NewSystemCheck(id string) *SystemCheck {
	return &SystemCheck{id: id}
}

func (c *SystemCheck) ID() string { return c.id }

func (c *SystemCheck) Collect() (SystemMetrics, error) {
	var m SystemMetrics

	cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return m, fmt.Errorf("cpu: %w", err)
	}
	if len(cpuPercent) > 0 {
		m.CPUPercent = cpuPercent[0]
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return m, fmt.Errorf("memory: %w", err)
	}
	m.MemPercent = memInfo.UsedPercent
	m.MemUsedGB = float64(memInfo.Used) / (1024 * 1024 * 1024)
	m.MemTotalGB = float64(memInfo.Total) / (1024 * 1024 * 1024)

	diskInfo, err := disk.Usage("/")
	if err != nil {
		return m, fmt.Errorf("disk: %w", err)
	}
	m.DiskPercent = diskInfo.UsedPercent
	m.DiskUsedGB = float64(diskInfo.Used) / (1024 * 1024 * 1024)
	m.DiskTotalGB = float64(diskInfo.Total) / (1024 * 1024 * 1024)

	return m, nil
}

func (c *SystemCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	metrics, err := c.Collect()
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("collect metrics: %v", err)
		return result
	}

	// Degraded if any metric > 90%
	if metrics.CPUPercent > 90 || metrics.MemPercent > 90 || metrics.DiskPercent > 90 {
		result.Status = StatusDegraded
		result.Error = fmt.Sprintf("high usage — cpu:%.0f%% mem:%.0f%% disk:%.0f%%",
			metrics.CPUPercent, metrics.MemPercent, metrics.DiskPercent)
		return result
	}

	result.Status = StatusHealthy
	return result
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/monitor/ -v -run TestSystem`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/monitor/system.go internal/monitor/system_test.go
git commit -m "feat: add system metrics check (CPU/mem/disk)"
```

---

### Task 5: Custom Shell Command Check

**Files:**
- Create: `internal/monitor/custom.go`
- Test: `internal/monitor/custom_test.go`

**Step 1: Write the failing test**

```go
// internal/monitor/custom_test.go
package monitor

import (
	"context"
	"testing"
	"time"
)

func TestCustomCheck_Success(t *testing.T) {
	c := NewCustomCheck("test-cmd", "echo hello", 5*time.Second)
	result := c.Check(context.Background())

	if result.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s (err: %s)", result.Status, result.Error)
	}
}

func TestCustomCheck_Failure(t *testing.T) {
	c := NewCustomCheck("test-cmd", "exit 1", 5*time.Second)
	result := c.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected down, got %s", result.Status)
	}
}

func TestCustomCheck_Timeout(t *testing.T) {
	c := NewCustomCheck("test-cmd", "sleep 10", 200*time.Millisecond)
	result := c.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected down on timeout, got %s", result.Status)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/monitor/ -v -run TestCustom`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/monitor/custom.go
package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type CustomCheck struct {
	id      string
	command string
	timeout time.Duration
}

func NewCustomCheck(id, command string, timeout time.Duration) *CustomCheck {
	return &CustomCheck{id: id, command: command, timeout: timeout}
}

func (c *CustomCheck) ID() string { return c.id }

func (c *CustomCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", c.command)
	output, err := cmd.CombinedOutput()
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("command failed: %v (output: %s)", err, string(output))
		return result
	}

	result.Status = StatusHealthy
	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/monitor/ -v -run TestCustom`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/monitor/custom.go internal/monitor/custom_test.go
git commit -m "feat: add custom shell command check"
```

---

### Task 6: Monitor Engine

**Files:**
- Create: `internal/monitor/engine.go`
- Test: `internal/monitor/engine_test.go`

**Step 1: Write the failing test**

```go
// internal/monitor/engine_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/monitor/ -v -run TestEngine`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/monitor/engine.go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/monitor/ -v -run TestEngine -race`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/monitor/engine.go internal/monitor/engine_test.go
git commit -m "feat: add concurrent monitor engine with goroutine-per-check"
```

---

### Task 7: Alert State Machine

**Files:**
- Create: `internal/alert/manager.go`
- Test: `internal/alert/manager_test.go`

**Step 1: Write the failing test**

```go
// internal/alert/manager_test.go
package alert

import (
	"testing"
)

func TestTransition_HealthyToDown(t *testing.T) {
	m := NewManager(3, 2)

	// 3 consecutive failures should transition to down
	for i := 0; i < 2; i++ {
		transition := m.Record("mon-1", false)
		if transition != nil {
			t.Errorf("unexpected transition on failure %d", i+1)
		}
	}

	transition := m.Record("mon-1", false)
	if transition == nil {
		t.Fatal("expected transition to down after 3 failures")
	}
	if transition.To != StatusDown {
		t.Errorf("expected down, got %s", transition.To)
	}
	if transition.From != StatusHealthy {
		t.Errorf("expected from healthy, got %s", transition.From)
	}
}

func TestTransition_DownToRecovered(t *testing.T) {
	m := NewManager(3, 2)

	// Go down first
	for i := 0; i < 3; i++ {
		m.Record("mon-1", false)
	}

	// 2 consecutive successes should recover
	transition := m.Record("mon-1", true)
	if transition != nil {
		t.Error("unexpected transition on first success")
	}

	transition = m.Record("mon-1", true)
	if transition == nil {
		t.Fatal("expected transition to recovered")
	}
	if transition.To != StatusRecovered {
		t.Errorf("expected recovered, got %s", transition.To)
	}
}

func TestTransition_FailureResetsSuccessCount(t *testing.T) {
	m := NewManager(3, 2)

	// Go down
	for i := 0; i < 3; i++ {
		m.Record("mon-1", false)
	}

	// One success then a failure
	m.Record("mon-1", true)
	m.Record("mon-1", false)

	// Next success should not recover (reset)
	transition := m.Record("mon-1", true)
	if transition != nil {
		t.Error("should not recover — success count was reset by failure")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/alert/ -v`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/alert/manager.go
package alert

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDown      Status = "down"
	StatusRecovered Status = "recovered"
)

type Transition struct {
	MonitorID string
	From      Status
	To        Status
}

type monitorState struct {
	status           Status
	consecutiveFails int
	consecutiveOK    int
}

type Manager struct {
	failThreshold    int
	recoverThreshold int
	states           map[string]*monitorState
}

func NewManager(failThreshold, recoverThreshold int) *Manager {
	return &Manager{
		failThreshold:    failThreshold,
		recoverThreshold: recoverThreshold,
		states:           make(map[string]*monitorState),
	}
}

func (m *Manager) Record(monitorID string, success bool) *Transition {
	state, ok := m.states[monitorID]
	if !ok {
		state = &monitorState{status: StatusHealthy}
		m.states[monitorID] = state
	}

	if success {
		state.consecutiveFails = 0
		state.consecutiveOK++
	} else {
		state.consecutiveOK = 0
		state.consecutiveFails++
	}

	switch state.status {
	case StatusHealthy, StatusRecovered:
		if state.consecutiveFails >= m.failThreshold {
			prev := state.status
			state.status = StatusDown
			state.consecutiveFails = 0
			return &Transition{MonitorID: monitorID, From: prev, To: StatusDown}
		}

	case StatusDown:
		if state.consecutiveOK >= m.recoverThreshold {
			state.status = StatusRecovered
			state.consecutiveOK = 0
			return &Transition{MonitorID: monitorID, From: StatusDown, To: StatusRecovered}
		}
	}

	return nil
}

func (m *Manager) GetStatus(monitorID string) Status {
	state, ok := m.states[monitorID]
	if !ok {
		return StatusHealthy
	}
	return state.status
}

func (m *Manager) Remove(monitorID string) {
	delete(m.states, monitorID)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/alert/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/alert/manager.go internal/alert/manager_test.go
git commit -m "feat: add alert state machine with configurable thresholds"
```

---

### Task 8: Alert Dispatchers (Visual, Desktop, Webhook)

**Files:**
- Create: `internal/alert/visual.go`
- Create: `internal/alert/desktop.go`
- Create: `internal/alert/webhook.go`
- Test: `internal/alert/webhook_test.go`

**Step 1: Write visual alert dispatcher**

```go
// internal/alert/visual.go
package alert

import "fmt"

type VisualDispatcher struct{}

func NewVisualDispatcher() *VisualDispatcher {
	return &VisualDispatcher{}
}

func (d *VisualDispatcher) Dispatch(t Transition) {
	// Terminal bell
	fmt.Print("\a")
}
```

**Step 2: Write desktop notification dispatcher**

Run first: `go get github.com/gen2brain/beeep`

```go
// internal/alert/desktop.go
package alert

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

type DesktopDispatcher struct{}

func NewDesktopDispatcher() *DesktopDispatcher {
	return &DesktopDispatcher{}
}

func (d *DesktopDispatcher) Dispatch(t Transition) {
	title := fmt.Sprintf("livepulse: %s", t.MonitorID)
	var body string
	switch t.To {
	case StatusDown:
		body = fmt.Sprintf("Monitor %s is DOWN", t.MonitorID)
	case StatusRecovered:
		body = fmt.Sprintf("Monitor %s has RECOVERED", t.MonitorID)
	default:
		body = fmt.Sprintf("Monitor %s status: %s", t.MonitorID, t.To)
	}
	_ = beeep.Notify(title, body, "")
}
```

**Step 3: Write webhook dispatcher with test**

```go
// internal/alert/webhook_test.go
package alert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookDispatcher_SendsPayload(t *testing.T) {
	var received WebhookPayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewWebhookDispatcher(srv.URL, 5*time.Second)
	d.Dispatch(Transition{
		MonitorID: "api-prod",
		From:      StatusHealthy,
		To:        StatusDown,
	})

	// Give goroutine time to fire
	time.Sleep(100 * time.Millisecond)

	if received.MonitorID != "api-prod" {
		t.Errorf("expected monitor api-prod, got %s", received.MonitorID)
	}
	if received.Status != string(StatusDown) {
		t.Errorf("expected status down, got %s", received.Status)
	}
}
```

```go
// internal/alert/webhook.go
package alert

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type WebhookPayload struct {
	MonitorID      string `json:"monitor"`
	Status         string `json:"status"`
	PreviousStatus string `json:"previous_status"`
	Timestamp      string `json:"timestamp"`
}

type WebhookDispatcher struct {
	url     string
	timeout time.Duration
	client  *http.Client
}

func NewWebhookDispatcher(url string, timeout time.Duration) *WebhookDispatcher {
	return &WebhookDispatcher{
		url:     url,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

func (d *WebhookDispatcher) Dispatch(t Transition) {
	payload := WebhookPayload{
		MonitorID:      t.MonitorID,
		Status:         string(t.To),
		PreviousStatus: string(t.From),
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Non-blocking: fire in goroutine to avoid stalling monitor loop
	go func() {
		resp, err := d.client.Post(d.url, "application/json", bytes.NewReader(data))
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}
```

**Step 4: Run tests**

Run: `go test ./internal/alert/ -v -race`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/alert/visual.go internal/alert/desktop.go internal/alert/webhook.go internal/alert/webhook_test.go
git commit -m "feat: add alert dispatchers (visual, desktop, webhook)"
```

---

### Task 9: TUI Styles

**Files:**
- Create: `internal/tui/styles.go`

**Step 1: Install TUI dependencies**

Run: `go get github.com/charmbracelet/bubbletea/v2 github.com/charmbracelet/lipgloss/v2 github.com/charmbracelet/bubbles/v2`

**Step 2: Write styles**

```go
// internal/tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss/v2"

var (
	// Colors
	colorHealthy  = lipgloss.Color("#00FF00")
	colorDegraded = lipgloss.Color("#FFD700")
	colorDown     = lipgloss.Color("#FF0000")
	colorMuted    = lipgloss.Color("#666666")
	colorAccent   = lipgloss.Color("#7D56F4")
	colorBg       = lipgloss.Color("#1a1a2e")
	colorFg       = lipgloss.Color("#FAFAFA")

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	// Status indicator styles
	healthyStyle  = lipgloss.NewStyle().Foreground(colorHealthy).Bold(true)
	degradedStyle = lipgloss.NewStyle().Foreground(colorDegraded).Bold(true)
	downStyle     = lipgloss.NewStyle().Foreground(colorDown).Bold(true)

	// Help bar
	helpStyle = lipgloss.NewStyle().Foreground(colorMuted)

	// Selected item
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333366")).
			Foreground(colorFg)

	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorFg).
			Background(colorAccent).
			Padding(0, 2).
			Width(80)

	// Label/value in detail view
	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(16)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorFg)
)
```

**Step 3: Commit**

```bash
git add internal/tui/styles.go
git commit -m "feat: add TUI style definitions"
```

---

### Task 10: TUI Components — Sparkline

**Files:**
- Create: `internal/tui/components/sparkline.go`
- Test: `internal/tui/components/sparkline_test.go`

**Step 1: Write the failing test**

```go
// internal/tui/components/sparkline_test.go
package components

import "testing"

func TestSparkline_Render(t *testing.T) {
	s := NewSparkline(20)
	s.Push(10)
	s.Push(50)
	s.Push(100)
	s.Push(25)

	output := s.View()
	if len(output) == 0 {
		t.Error("expected non-empty sparkline output")
	}
}

func TestSparkline_MaxWidth(t *testing.T) {
	s := NewSparkline(5)
	for i := 0; i < 10; i++ {
		s.Push(float64(i * 10))
	}

	// Should only keep last 5 values
	if len(s.values) != 5 {
		t.Errorf("expected 5 values, got %d", len(s.values))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/components/ -v -run TestSparkline`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/tui/components/sparkline.go
package components

import "strings"

var blocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

type Sparkline struct {
	values   []float64
	maxWidth int
}

func NewSparkline(maxWidth int) *Sparkline {
	return &Sparkline{maxWidth: maxWidth}
}

func (s *Sparkline) Push(val float64) {
	s.values = append(s.values, val)
	if len(s.values) > s.maxWidth {
		s.values = s.values[len(s.values)-s.maxWidth:]
	}
}

func (s *Sparkline) View() string {
	if len(s.values) == 0 {
		return strings.Repeat(" ", s.maxWidth)
	}

	min, max := s.values[0], s.values[0]
	for _, v := range s.values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var sb strings.Builder
	span := max - min
	for _, v := range s.values {
		idx := 0
		if span > 0 {
			idx = int((v - min) / span * float64(len(blocks)-1))
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}

	// Pad to maxWidth
	for sb.Len() < s.maxWidth {
		sb.WriteRune(' ')
	}

	return sb.String()
}

func (s *Sparkline) Clear() {
	s.values = nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/components/ -v -run TestSparkline`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/components/sparkline.go internal/tui/components/sparkline_test.go
git commit -m "feat: add sparkline component for latency graphs"
```

---

### Task 11: TUI Components — Status Bar + Event Log

**Files:**
- Create: `internal/tui/components/statusbar.go`
- Create: `internal/tui/components/eventlog.go`

**Step 1: Write status bar**

```go
// internal/tui/components/statusbar.go
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
)

type StatusBar struct {
	healthy  int
	degraded int
	down     int
	total    int
	width    int
}

func NewStatusBar(width int) *StatusBar {
	return &StatusBar{width: width}
}

func (s *StatusBar) Update(healthy, degraded, down int) {
	s.healthy = healthy
	s.degraded = degraded
	s.down = down
	s.total = healthy + degraded + down
}

func (s *StatusBar) View() string {
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	return fmt.Sprintf("%s %s  %s %s  %s %s  %s %s",
		greenStyle.Render("●"), greenStyle.Render(fmt.Sprintf("%d healthy", s.healthy)),
		yellowStyle.Render("●"), yellowStyle.Render(fmt.Sprintf("%d degraded", s.degraded)),
		redStyle.Render("●"), redStyle.Render(fmt.Sprintf("%d down", s.down)),
		mutedStyle.Render("|"), mutedStyle.Render(fmt.Sprintf("%d total", s.total)),
	)
}
```

**Step 2: Write event log**

```go
// internal/tui/components/eventlog.go
package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
)

type Event struct {
	Time    time.Time
	Monitor string
	Message string
	IsError bool
}

type EventLog struct {
	events   []Event
	maxLines int
}

func NewEventLog(maxLines int) *EventLog {
	return &EventLog{maxLines: maxLines}
}

func (e *EventLog) Add(monitor, message string, isError bool) {
	e.events = append(e.events, Event{
		Time:    time.Now(),
		Monitor: monitor,
		Message: message,
		IsError: isError,
	})
	if len(e.events) > e.maxLines {
		e.events = e.events[len(e.events)-e.maxLines:]
	}
}

func (e *EventLog) View() string {
	if len(e.events) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render("  No events yet...")
	}

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	var result string
	// Show latest events at bottom
	start := 0
	if len(e.events) > e.maxLines {
		start = len(e.events) - e.maxLines
	}

	for _, ev := range e.events[start:] {
		ts := timeStyle.Render(ev.Time.Format("15:04:05"))
		style := okStyle
		if ev.IsError {
			style = errStyle
		}
		line := fmt.Sprintf("  %s  %s  %s\n", ts, style.Render(ev.Monitor), ev.Message)
		result += line
	}

	return result
}

func (e *EventLog) Clear() {
	e.events = nil
}
```

**Step 3: Commit**

```bash
git add internal/tui/components/statusbar.go internal/tui/components/eventlog.go
git commit -m "feat: add status bar and event log TUI components"
```

---

### Task 12: TUI Dashboard View

**Files:**
- Create: `internal/tui/dashboard.go`

This is the main dashboard rendering — the list of monitors with status, latency sparklines, and the event log panel.

**Step 1: Write dashboard view**

```go
// internal/tui/dashboard.go
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/v2"

	"github.com/arjunmyanger/livepulse/internal/monitor"
	"github.com/arjunmyanger/livepulse/internal/tui/components"
)

type MonitorView struct {
	ID        string
	Name      string
	Type      string
	Target    string
	Status    monitor.Status
	Latency   time.Duration
	LastCheck time.Time
	Error     string
	Sparkline *components.Sparkline
}

func renderDashboard(monitors []MonitorView, selected int, statusBar *components.StatusBar, eventLog *components.EventLog, width, height int) string {
	var b strings.Builder

	// Header
	header := headerStyle.Width(width).Render(" livepulse — Real-Time Infrastructure Monitor")
	b.WriteString(header + "\n\n")

	// Status bar
	b.WriteString("  " + statusBar.View() + "\n\n")

	// Monitor list
	if len(monitors) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(2, 4).
			Render("No monitors configured. Press 'a' to add one.")
		b.WriteString(empty + "\n")
	} else {
		for i, m := range monitors {
			line := renderMonitorLine(m, i == selected, width-4)
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")

	// Event log panel
	logTitle := titleStyle.Render("Event Log")
	b.WriteString(logTitle + "\n")
	b.WriteString(eventLog.View())
	b.WriteString("\n")

	// Help bar
	help := helpStyle.Render("  a:add  e:edit  d:delete  enter:detail  ?:help  q:quit")
	b.WriteString(help)

	return b.String()
}

func renderMonitorLine(m MonitorView, selected bool, width int) string {
	// Status indicator
	var statusIcon string
	var statusStyle lipgloss.Style
	switch m.Status {
	case monitor.StatusHealthy:
		statusIcon = "●"
		statusStyle = healthyStyle
	case monitor.StatusDegraded:
		statusIcon = "●"
		statusStyle = degradedStyle
	case monitor.StatusDown:
		statusIcon = "●"
		statusStyle = downStyle
	default:
		statusIcon = "○"
		statusStyle = lipgloss.NewStyle().Foreground(colorMuted)
	}

	icon := statusStyle.Render(statusIcon)

	name := m.Name
	if len(name) > 20 {
		name = name[:17] + "..."
	}

	latencyStr := fmt.Sprintf("%dms", m.Latency.Milliseconds())
	sparkline := ""
	if m.Sparkline != nil {
		sparkline = m.Sparkline.View()
	}

	typeLabel := lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("[%s]", m.Type))

	line := fmt.Sprintf("  %s  %-20s %s  %8s  %s", icon, name, typeLabel, latencyStr, sparkline)

	if selected {
		line = selectedStyle.Render(line)
	}

	return line
}
```

**Step 2: Commit**

```bash
git add internal/tui/dashboard.go
git commit -m "feat: add dashboard view with monitor list and event log"
```

---

### Task 13: TUI Detail View

**Files:**
- Create: `internal/tui/detail.go`

**Step 1: Write detail view**

```go
// internal/tui/detail.go
package tui

import (
	"fmt"
	"strings"

	"github.com/arjunmyanger/livepulse/internal/monitor"
	"github.com/arjunmyanger/livepulse/internal/tui/components"
	"github.com/charmbracelet/lipgloss/v2"
)

func renderDetail(m MonitorView, results []monitor.CheckResult, sparkline *components.Sparkline, width int) string {
	var b strings.Builder

	// Header
	header := headerStyle.Width(width).Render(fmt.Sprintf(" Monitor: %s", m.Name))
	b.WriteString(header + "\n\n")

	// Status
	var statusStyle lipgloss.Style
	switch m.Status {
	case monitor.StatusHealthy:
		statusStyle = healthyStyle
	case monitor.StatusDegraded:
		statusStyle = degradedStyle
	case monitor.StatusDown:
		statusStyle = downStyle
	default:
		statusStyle = lipgloss.NewStyle().Foreground(colorMuted)
	}

	row := func(label, value string) string {
		return fmt.Sprintf("  %s %s\n", labelStyle.Render(label), valueStyle.Render(value))
	}

	b.WriteString(row("Status:", statusStyle.Render(string(m.Status))))
	b.WriteString(row("Type:", m.Type))
	b.WriteString(row("Target:", m.Target))
	b.WriteString(row("Latency:", fmt.Sprintf("%dms", m.Latency.Milliseconds())))
	b.WriteString(row("Last Check:", m.LastCheck.Format("15:04:05")))

	if m.Error != "" {
		errStyle := lipgloss.NewStyle().Foreground(colorDown)
		b.WriteString(row("Error:", errStyle.Render(m.Error)))
	}

	b.WriteString("\n")

	// Latency graph
	graphTitle := titleStyle.Render("Latency History")
	b.WriteString("  " + graphTitle + "\n")
	if sparkline != nil {
		b.WriteString("  " + sparkline.View() + "\n")
	}

	b.WriteString("\n")

	// Recent results
	recentTitle := titleStyle.Render("Recent Checks")
	b.WriteString("  " + recentTitle + "\n")

	timeStyle := lipgloss.NewStyle().Foreground(colorMuted)
	count := len(results)
	start := 0
	if count > 10 {
		start = count - 10
	}
	for _, r := range results[start:] {
		var sIcon string
		switch r.Status {
		case monitor.StatusHealthy:
			sIcon = healthyStyle.Render("●")
		case monitor.StatusDegraded:
			sIcon = degradedStyle.Render("●")
		default:
			sIcon = downStyle.Render("●")
		}
		ts := timeStyle.Render(r.Timestamp.Format("15:04:05"))
		errStr := ""
		if r.Error != "" {
			errStr = lipgloss.NewStyle().Foreground(colorDown).Render(" — " + r.Error)
		}
		b.WriteString(fmt.Sprintf("  %s %s  %dms%s\n", sIcon, ts, r.Latency.Milliseconds(), errStr))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  esc:back  e:edit  d:delete"))

	return b.String()
}
```

**Step 2: Commit**

```bash
git add internal/tui/detail.go
git commit -m "feat: add monitor detail view with history and sparkline"
```

---

### Task 14: TUI Setup Form (Add/Edit Monitors)

**Files:**
- Create: `internal/tui/setup.go`

This uses bubbles v2 textinput for form fields. The setup form is a multi-step wizard: type -> target -> name -> interval -> alerts.

**Step 1: Write setup form**

```go
// internal/tui/setup.go
package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"github.com/arjunmyanger/livepulse/internal/store"
)

type setupStep int

const (
	stepType setupStep = iota
	stepTarget
	stepName
	stepInterval
	stepAlertVisual
	stepAlertDesktop
	stepAlertWebhook
	stepConfirm
)

type setupModel struct {
	step       setupStep
	monType    string
	typeIdx    int
	target     textinput.Model
	name       textinput.Model
	interval   textinput.Model
	webhook    textinput.Model
	alertVis   bool
	alertDesk  bool
	editing    *store.MonitorConfig // nil for new, set for edit
	width      int
}

var monitorTypes = []string{"http", "tcp", "system", "custom"}

func newSetupModel(existing *store.MonitorConfig) setupModel {
	target := textinput.New()
	target.Placeholder = "https://example.com/health"
	target.SetWidth(50)
	target.Focus()

	name := textinput.New()
	name.Placeholder = "My Monitor"
	name.SetWidth(50)

	interval := textinput.New()
	interval.Placeholder = "10"
	interval.SetWidth(10)

	webhook := textinput.New()
	webhook.Placeholder = "https://hooks.slack.com/..."
	webhook.SetWidth(50)

	m := setupModel{
		step:      stepType,
		typeIdx:   0,
		target:    target,
		name:      name,
		interval:  interval,
		webhook:   webhook,
		alertVis:  true,
		alertDesk: false,
		editing:   existing,
	}

	if existing != nil {
		for i, t := range monitorTypes {
			if t == existing.Type {
				m.typeIdx = i
				break
			}
		}
		m.monType = existing.Type
		m.target.SetValue(existing.Target)
		m.name.SetValue(existing.Name)
		m.interval.SetValue(strconv.Itoa(existing.Interval))
		m.alertVis = existing.Alerts.Visual
		m.alertDesk = existing.Alerts.Desktop
		m.webhook.SetValue(existing.Alerts.Webhook)
	}

	return m
}

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch m.step {
		case stepType:
			switch msg.String() {
			case "up", "k":
				if m.typeIdx > 0 {
					m.typeIdx--
				}
			case "down", "j":
				if m.typeIdx < len(monitorTypes)-1 {
					m.typeIdx++
				}
			case "enter":
				m.monType = monitorTypes[m.typeIdx]
				m.step = stepTarget
				if m.monType == "system" {
					m.target.SetValue("localhost")
					m.step = stepName
				}
				m.target.Focus()
				return m, m.target.Focus()
			}
			return m, nil

		case stepTarget:
			switch msg.String() {
			case "enter":
				if m.target.Value() != "" {
					m.step = stepName
					m.target.Blur()
					return m, m.name.Focus()
				}
			}
			var cmd tea.Cmd
			m.target, cmd = m.target.Update(msg)
			return m, cmd

		case stepName:
			switch msg.String() {
			case "enter":
				if m.name.Value() != "" {
					m.step = stepInterval
					m.name.Blur()
					return m, m.interval.Focus()
				}
			}
			var cmd tea.Cmd
			m.name, cmd = m.name.Update(msg)
			return m, cmd

		case stepInterval:
			switch msg.String() {
			case "enter":
				if m.interval.Value() != "" {
					m.step = stepAlertVisual
					m.interval.Blur()
				}
			}
			var cmd tea.Cmd
			m.interval, cmd = m.interval.Update(msg)
			return m, cmd

		case stepAlertVisual:
			switch msg.String() {
			case "y", "Y":
				m.alertVis = true
				m.step = stepAlertDesktop
			case "n", "N":
				m.alertVis = false
				m.step = stepAlertDesktop
			case "enter":
				m.step = stepAlertDesktop
			}
			return m, nil

		case stepAlertDesktop:
			switch msg.String() {
			case "y", "Y":
				m.alertDesk = true
				m.step = stepAlertWebhook
				return m, m.webhook.Focus()
			case "n", "N":
				m.alertDesk = false
				m.step = stepAlertWebhook
				return m, m.webhook.Focus()
			case "enter":
				m.step = stepAlertWebhook
				return m, m.webhook.Focus()
			}
			return m, nil

		case stepAlertWebhook:
			switch msg.String() {
			case "enter":
				m.webhook.Blur()
				m.step = stepConfirm
			}
			var cmd tea.Cmd
			m.webhook, cmd = m.webhook.Update(msg)
			return m, cmd

		case stepConfirm:
			// Handled by parent model
			return m, nil
		}
	}

	return m, nil
}

func (m setupModel) View() string {
	var b strings.Builder

	action := "Add New Monitor"
	if m.editing != nil {
		action = "Edit Monitor"
	}
	header := headerStyle.Width(m.width).Render(fmt.Sprintf(" %s", action))
	b.WriteString(header + "\n\n")

	activeStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	pendingStyle := lipgloss.NewStyle().Foreground(colorMuted)

	stepLabel := func(s setupStep, label string) string {
		if m.step == s {
			return activeStyle.Render("→ " + label)
		} else if m.step > s {
			return doneStyle.Render("✓ " + label)
		}
		return pendingStyle.Render("  " + label)
	}

	// Type selection
	b.WriteString(stepLabel(stepType, "Monitor Type") + "\n")
	if m.step == stepType {
		for i, t := range monitorTypes {
			cursor := "  "
			if i == m.typeIdx {
				cursor = activeStyle.Render("> ")
			}
			b.WriteString(fmt.Sprintf("    %s%s\n", cursor, t))
		}
	} else if m.step > stepType {
		b.WriteString(fmt.Sprintf("    %s\n", m.monType))
	}

	// Target
	b.WriteString(stepLabel(stepTarget, "Target") + "\n")
	if m.step == stepTarget {
		b.WriteString("    " + m.target.View() + "\n")
	} else if m.step > stepTarget {
		b.WriteString(fmt.Sprintf("    %s\n", m.target.Value()))
	}

	// Name
	b.WriteString(stepLabel(stepName, "Name") + "\n")
	if m.step == stepName {
		b.WriteString("    " + m.name.View() + "\n")
	} else if m.step > stepName {
		b.WriteString(fmt.Sprintf("    %s\n", m.name.Value()))
	}

	// Interval
	b.WriteString(stepLabel(stepInterval, "Check Interval (seconds)") + "\n")
	if m.step == stepInterval {
		b.WriteString("    " + m.interval.View() + "\n")
	} else if m.step > stepInterval {
		b.WriteString(fmt.Sprintf("    %ss\n", m.interval.Value()))
	}

	// Alert visual
	b.WriteString(stepLabel(stepAlertVisual, "Visual Alerts (y/n)") + "\n")
	if m.step == stepAlertVisual {
		b.WriteString(fmt.Sprintf("    Current: %v — press y/n\n", m.alertVis))
	} else if m.step > stepAlertVisual {
		b.WriteString(fmt.Sprintf("    %v\n", m.alertVis))
	}

	// Alert desktop
	b.WriteString(stepLabel(stepAlertDesktop, "Desktop Notifications (y/n)") + "\n")
	if m.step == stepAlertDesktop {
		b.WriteString(fmt.Sprintf("    Current: %v — press y/n\n", m.alertDesk))
	} else if m.step > stepAlertDesktop {
		b.WriteString(fmt.Sprintf("    %v\n", m.alertDesk))
	}

	// Webhook
	b.WriteString(stepLabel(stepAlertWebhook, "Webhook URL (optional)") + "\n")
	if m.step == stepAlertWebhook {
		b.WriteString("    " + m.webhook.View() + "\n")
	} else if m.step > stepAlertWebhook {
		url := m.webhook.Value()
		if url == "" {
			url = "(none)"
		}
		b.WriteString(fmt.Sprintf("    %s\n", url))
	}

	// Confirm
	if m.step == stepConfirm {
		b.WriteString("\n")
		b.WriteString(activeStyle.Render("  Press enter to save, esc to cancel") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  esc:cancel"))

	return b.String()
}

func (m setupModel) toConfig(id string) store.MonitorConfig {
	intervalSec, _ := strconv.Atoi(m.interval.Value())
	if intervalSec <= 0 {
		intervalSec = 10
	}

	return store.MonitorConfig{
		ID:       id,
		Name:     m.name.Value(),
		Type:     m.monType,
		Target:   m.target.Value(),
		Interval: intervalSec,
		Alerts: store.AlertConfig{
			Visual:  m.alertVis,
			Desktop: m.alertDesk,
			Webhook: m.webhook.Value(),
		},
	}
}
```

**Step 2: Commit**

```bash
git add internal/tui/setup.go
git commit -m "feat: add interactive setup wizard for adding/editing monitors"
```

---

### Task 15: TUI Main App Model

**Files:**
- Create: `internal/tui/app.go`

This ties everything together: the bubbletea main model that handles views (dashboard/detail/setup), receives check results via tea.Cmd subscriptions, and manages the monitor engine.

**Step 1: Write main app model**

```go
// internal/tui/app.go
package tui

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/arjunmyanger/livepulse/internal/alert"
	"github.com/arjunmyanger/livepulse/internal/monitor"
	"github.com/arjunmyanger/livepulse/internal/store"
	"github.com/arjunmyanger/livepulse/internal/tui/components"
)

type view int

const (
	viewDashboard view = iota
	viewDetail
	viewSetup
	viewHelp
)

type checkResultMsg monitor.CheckResult

type App struct {
	// State
	view     view
	monitors []MonitorView
	selected int
	results  map[string][]monitor.CheckResult
	width    int
	height   int

	// Components
	statusBar *components.StatusBar
	eventLog  *components.EventLog

	// Engine
	engine    *monitor.Engine
	resultsCh chan monitor.CheckResult

	// Alerts
	alertMgr     *alert.Manager
	visualAlert  *alert.VisualDispatcher
	desktopAlert *alert.DesktopDispatcher
	webhooks     map[string]*alert.WebhookDispatcher

	// Store
	store   *store.Store
	configs []store.MonitorConfig

	// Sub-views
	setup   setupModel
	detailID string
}

func NewApp(storePath string) *App {
	resultsCh := make(chan monitor.CheckResult, 100)
	return &App{
		view:         viewDashboard,
		monitors:     []MonitorView{},
		results:      make(map[string][]monitor.CheckResult),
		statusBar:    components.NewStatusBar(80),
		eventLog:     components.NewEventLog(10),
		engine:       monitor.NewEngine(resultsCh),
		resultsCh:    resultsCh,
		alertMgr:     alert.NewManager(3, 2),
		visualAlert:  alert.NewVisualDispatcher(),
		desktopAlert: alert.NewDesktopDispatcher(),
		webhooks:     make(map[string]*alert.WebhookDispatcher),
		store:        store.New(storePath),
	}
}

func DefaultStorePath() string {
	home := ""
	// Use os.UserHomeDir at call site
	return filepath.Join(home, ".livepulse", "monitors.json")
}

func (a *App) Init() tea.Cmd {
	// Load saved monitors
	configs, err := a.store.Load()
	if err == nil {
		a.configs = configs
		for _, cfg := range configs {
			a.addMonitorFromConfig(cfg)
		}
	}

	return tea.Batch(
		a.waitForResult(),
		a.tickCmd(),
	)
}

func (a *App) waitForResult() tea.Cmd {
	return func() tea.Msg {
		result := <-a.resultsCh
		return checkResultMsg(result)
	}
}

type tickMsg time.Time

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case checkResultMsg:
		a.handleCheckResult(monitor.CheckResult(msg))
		return a, a.waitForResult()

	case tickMsg:
		return a, a.tickCmd()

	case tea.KeyPressMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		}

		switch a.view {
		case viewDashboard:
			return a.updateDashboard(msg)
		case viewDetail:
			return a.updateDetail(msg)
		case viewSetup:
			return a.updateSetup(msg)
		case viewHelp:
			return a.updateHelp(msg)
		}
	}

	return a, nil
}

func (a *App) updateDashboard(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "up", "k":
		if a.selected > 0 {
			a.selected--
		}
	case "down", "j":
		if a.selected < len(a.monitors)-1 {
			a.selected++
		}
	case "enter":
		if len(a.monitors) > 0 {
			a.detailID = a.monitors[a.selected].ID
			a.view = viewDetail
		}
	case "a":
		a.setup = newSetupModel(nil)
		a.setup.width = a.width
		a.view = viewSetup
	case "e":
		if len(a.monitors) > 0 {
			id := a.monitors[a.selected].ID
			for _, cfg := range a.configs {
				if cfg.ID == id {
					a.setup = newSetupModel(&cfg)
					a.setup.width = a.width
					a.view = viewSetup
					break
				}
			}
		}
	case "d":
		if len(a.monitors) > 0 {
			a.deleteMonitor(a.monitors[a.selected].ID)
			if a.selected >= len(a.monitors) && a.selected > 0 {
				a.selected--
			}
		}
	case "?":
		a.view = viewHelp
	}
	return a, nil
}

func (a *App) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.view = viewDashboard
	case "e":
		for _, cfg := range a.configs {
			if cfg.ID == a.detailID {
				a.setup = newSetupModel(&cfg)
				a.setup.width = a.width
				a.view = viewSetup
				break
			}
		}
	case "d":
		a.deleteMonitor(a.detailID)
		a.view = viewDashboard
	}
	return a, nil
}

func (a *App) updateSetup(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.view = viewDashboard
		return a, nil
	case "enter":
		if a.setup.step == stepConfirm {
			a.saveFromSetup()
			a.view = viewDashboard
			return a, nil
		}
	}

	var cmd tea.Cmd
	a.setup, cmd = a.setup.Update(msg)
	return a, cmd
}

func (a *App) updateHelp(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "q":
		a.view = viewDashboard
	}
	return a, nil
}

func (a *App) View() string {
	switch a.view {
	case viewDashboard:
		return renderDashboard(a.monitors, a.selected, a.statusBar, a.eventLog, a.width, a.height)
	case viewDetail:
		var mv MonitorView
		for _, m := range a.monitors {
			if m.ID == a.detailID {
				mv = m
				break
			}
		}
		return renderDetail(mv, a.results[a.detailID], mv.Sparkline, a.width)
	case viewSetup:
		return a.setup.View()
	case viewHelp:
		return renderHelp(a.width)
	}
	return ""
}

func renderHelp(width int) string {
	header := headerStyle.Width(width).Render(" livepulse — Help")
	body := `
  Keyboard Shortcuts:

  Dashboard:
    a         Add new monitor
    e         Edit selected monitor
    d         Delete selected monitor
    enter     View monitor details
    j/k       Navigate up/down
    ?         Toggle help
    q         Quit

  Detail View:
    e         Edit monitor
    d         Delete monitor
    esc       Back to dashboard

  Setup Wizard:
    j/k       Navigate options
    enter     Confirm / next step
    y/n       Toggle yes/no options
    esc       Cancel

  Press esc to close this help.
`
	return header + "\n" + body
}

func (a *App) handleCheckResult(result monitor.CheckResult) {
	// Store result
	a.results[result.MonitorID] = append(a.results[result.MonitorID], result)
	if len(a.results[result.MonitorID]) > 100 {
		a.results[result.MonitorID] = a.results[result.MonitorID][1:]
	}

	// Update monitor view
	for i, m := range a.monitors {
		if m.ID == result.MonitorID {
			a.monitors[i].Status = result.Status
			a.monitors[i].Latency = result.Latency
			a.monitors[i].LastCheck = result.Timestamp
			a.monitors[i].Error = result.Error
			if a.monitors[i].Sparkline != nil {
				a.monitors[i].Sparkline.Push(float64(result.Latency.Milliseconds()))
			}
			break
		}
	}

	// Update status bar counts
	healthy, degraded, down := 0, 0, 0
	for _, m := range a.monitors {
		switch m.Status {
		case monitor.StatusHealthy:
			healthy++
		case monitor.StatusDegraded:
			degraded++
		case monitor.StatusDown:
			down++
		}
	}
	a.statusBar.Update(healthy, degraded, down)

	// Alert state machine
	success := result.Status == monitor.StatusHealthy
	transition := a.alertMgr.Record(result.MonitorID, success)

	if transition != nil {
		// Log event
		isErr := transition.To == alert.StatusDown
		a.eventLog.Add(transition.MonitorID, fmt.Sprintf("%s -> %s", transition.From, transition.To), isErr)

		// Find config for this monitor
		for _, cfg := range a.configs {
			if cfg.ID == result.MonitorID {
				if cfg.Alerts.Visual {
					a.visualAlert.Dispatch(*transition)
				}
				if cfg.Alerts.Desktop {
					a.desktopAlert.Dispatch(*transition)
				}
				if cfg.Alerts.Webhook != "" {
					if wh, ok := a.webhooks[cfg.ID]; ok {
						wh.Dispatch(*transition)
					}
				}
				break
			}
		}
	}
}

func (a *App) addMonitorFromConfig(cfg store.MonitorConfig) {
	var checker monitor.Checker
	interval := time.Duration(cfg.Interval) * time.Second
	timeout := 5 * time.Second

	switch cfg.Type {
	case "http":
		expectedStatus := cfg.ExpectedStatus
		if expectedStatus == 0 {
			expectedStatus = 200
		}
		checker = monitor.NewHTTPCheck(cfg.ID, cfg.Target, expectedStatus, timeout)
	case "tcp":
		checker = monitor.NewTCPCheck(cfg.ID, cfg.Target, timeout)
	case "system":
		checker = monitor.NewSystemCheck(cfg.ID)
	case "custom":
		checker = monitor.NewCustomCheck(cfg.ID, cfg.Command, timeout)
	}

	if checker != nil {
		a.engine.Add(checker, interval)
	}

	// Setup webhook dispatcher if configured
	if cfg.Alerts.Webhook != "" {
		a.webhooks[cfg.ID] = alert.NewWebhookDispatcher(cfg.Alerts.Webhook, 10*time.Second)
	}

	// Add to view list
	a.monitors = append(a.monitors, MonitorView{
		ID:        cfg.ID,
		Name:      cfg.Name,
		Type:      cfg.Type,
		Target:    cfg.Target,
		Status:    monitor.StatusHealthy,
		Sparkline: components.NewSparkline(30),
	})
}

func (a *App) deleteMonitor(id string) {
	a.engine.Remove(id)
	a.alertMgr.Remove(id)
	delete(a.results, id)
	delete(a.webhooks, id)

	// Remove from monitors view
	for i, m := range a.monitors {
		if m.ID == id {
			a.monitors = append(a.monitors[:i], a.monitors[i+1:]...)
			break
		}
	}

	// Remove from configs
	for i, cfg := range a.configs {
		if cfg.ID == id {
			a.configs = append(a.configs[:i], a.configs[i+1:]...)
			break
		}
	}

	a.store.Save(a.configs)
}

func (a *App) saveFromSetup() {
	id := fmt.Sprintf("mon-%d", time.Now().UnixNano())
	if a.setup.editing != nil {
		id = a.setup.editing.ID
		a.deleteMonitor(id)
	}

	cfg := a.setup.toConfig(id)
	a.configs = append(a.configs, cfg)
	a.addMonitorFromConfig(cfg)
	a.store.Save(a.configs)
}
```

**Step 2: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: add main TUI app model tying all components together"
```

---

### Task 16: Entry Point — main.go

**Files:**
- Create: `cmd/livepulse/main.go`

**Step 1: Write main.go**

```go
// cmd/livepulse/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/arjunmyanger/livepulse/internal/tui"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	storePath := filepath.Join(home, ".livepulse", "monitors.json")
	app := tui.NewApp(storePath)

	// Start the monitor engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.StartEngine(ctx)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Add StartEngine method to App**

Add to `internal/tui/app.go`:

```go
func (a *App) StartEngine(ctx context.Context) {
	a.engine.Start(ctx)
}
```

**Step 3: Run build**

Run: `go mod tidy && go build -o livepulse ./cmd/livepulse`
Expected: Compiles successfully

**Step 4: Commit**

```bash
git add cmd/livepulse/main.go internal/tui/app.go
git commit -m "feat: add entry point and wire up engine startup"
```

---

### Task 17: Integration Test — Full Smoke Test

**Files:**
- Create: `cmd/livepulse/main_test.go`

**Step 1: Write smoke test**

```go
// cmd/livepulse/main_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/arjunmyanger/livepulse/internal/monitor"
	"github.com/arjunmyanger/livepulse/internal/store"
)

func TestEndToEnd_MonitorHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dir := t.TempDir()
	s := store.New(filepath.Join(dir, "monitors.json"))

	cfg := store.MonitorConfig{
		ID:       "e2e-http",
		Name:     "E2E Test",
		Type:     "http",
		Target:   srv.URL,
		Interval: 1,
		Alerts:   store.AlertConfig{Visual: true},
	}
	s.Save([]store.MonitorConfig{cfg})

	// Verify the check works
	checker := monitor.NewHTTPCheck(cfg.ID, cfg.Target, 200, 5*time.Second)
	result := checker.Check(t.Context())

	if result.Status != monitor.StatusHealthy {
		t.Errorf("expected healthy, got %s (err: %s)", result.Status, result.Error)
	}
}
```

**Step 2: Run all tests**

Run: `go test ./... -v -race`
Expected: ALL PASS

**Step 3: Final commit**

```bash
git add cmd/livepulse/main_test.go
git commit -m "feat: add end-to-end smoke test"
```

---

## Build Order Summary

| Task | Component | Dependencies |
|------|-----------|-------------|
| 1 | Store (persistence) | None |
| 2 | Check interface + HTTP check | None |
| 3 | TCP check | Task 2 (interface) |
| 4 | System metrics check | Task 2 (interface) |
| 5 | Custom command check | Task 2 (interface) |
| 6 | Monitor engine | Task 2 (interface) |
| 7 | Alert state machine | None |
| 8 | Alert dispatchers | Task 7 |
| 9 | TUI styles | None |
| 10 | Sparkline component | None |
| 11 | Status bar + event log | None |
| 12 | Dashboard view | Tasks 9-11 |
| 13 | Detail view | Tasks 9-11 |
| 14 | Setup wizard | Task 9 |
| 15 | Main app model | Tasks 1, 6-8, 12-14 |
| 16 | Entry point (main.go) | Task 15 |
| 17 | Smoke test | Tasks 1-2, 16 |

**Parallelizable groups:**
- Tasks 1-2, 7, 9-11 can all be built in parallel (no deps)
- Tasks 3-6 can be parallel (only depend on Task 2)
- Tasks 8, 12-14 can be parallel (depend on earlier tasks)
