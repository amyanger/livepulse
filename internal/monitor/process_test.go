package monitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// testProcessName returns the name of the current test binary,
// which is guaranteed to be running during tests.
func testProcessName() string {
	exe, _ := os.Executable()
	return filepath.Base(exe)
}

func TestProcessCheck_Down(t *testing.T) {
	c := NewProcessCheck("test-proc", "livepulse_nonexistent_xyz_999", 80, 80)
	result := c.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected down for nonexistent process, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for nonexistent process")
	}
	if result.MonitorID != "test-proc" {
		t.Errorf("expected monitor ID test-proc, got %s", result.MonitorID)
	}
}

func TestProcessCheck_Healthy(t *testing.T) {
	// The test binary itself is always running
	name := testProcessName()
	c := NewProcessCheck("test-proc", name, 80, 80)
	result := c.Check(context.Background())

	if result.Status != StatusHealthy && result.Status != StatusDegraded {
		t.Errorf("expected healthy or degraded for own process, got %s (err: %s)", result.Status, result.Error)
	}
}

func TestProcessCheck_Degraded(t *testing.T) {
	// Thresholds of 0 mean any usage triggers degraded
	name := testProcessName()
	c := NewProcessCheck("test-proc", name, 0, 0)
	result := c.Check(context.Background())

	if result.Status != StatusDegraded {
		t.Errorf("expected degraded with 0%% thresholds, got %s", result.Status)
	}
}

func TestProcessCheck_Collect(t *testing.T) {
	name := testProcessName()
	c := NewProcessCheck("test-proc", name, 80, 80)
	metrics, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if metrics.PID <= 0 {
		t.Errorf("expected positive PID, got %d", metrics.PID)
	}
	if metrics.CPUPercent < 0 {
		t.Errorf("cpu percent should be >= 0, got %f", metrics.CPUPercent)
	}
	if metrics.MemPercent < 0 || metrics.MemPercent > 100 {
		t.Errorf("mem percent out of range: %f", metrics.MemPercent)
	}
}
