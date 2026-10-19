package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/amyanger/livepulse/internal/monitor"
	"github.com/amyanger/livepulse/internal/store"
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
	if err := s.Save([]store.MonitorConfig{cfg}); err != nil {
		t.Fatalf("save: %v", err)
	}

	checker := monitor.NewHTTPCheck(cfg.ID, cfg.Target, 200, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result := checker.Check(ctx)

	if result.Status != monitor.StatusHealthy {
		t.Errorf("expected healthy, got %s (err: %s)", result.Status, result.Error)
	}
}
