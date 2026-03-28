package store

import (
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
