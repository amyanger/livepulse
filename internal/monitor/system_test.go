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
