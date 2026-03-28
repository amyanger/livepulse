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
