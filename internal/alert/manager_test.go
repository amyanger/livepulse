package alert

import (
	"testing"
)

func TestTransition_HealthyToDown(t *testing.T) {
	m := NewManager(3, 2)

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

	for i := 0; i < 3; i++ {
		m.Record("mon-1", false)
	}

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

	for i := 0; i < 3; i++ {
		m.Record("mon-1", false)
	}

	m.Record("mon-1", true)
	m.Record("mon-1", false)

	transition := m.Record("mon-1", true)
	if transition != nil {
		t.Error("should not recover — success count was reset by failure")
	}
}
