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
