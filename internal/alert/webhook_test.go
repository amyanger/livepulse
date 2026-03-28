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
	done := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
		close(done)
	}))
	defer srv.Close()

	d := NewWebhookDispatcher(srv.URL, 5*time.Second)
	d.Dispatch(Transition{
		MonitorID: "api-prod",
		From:      StatusHealthy,
		To:        StatusDown,
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook request")
	}

	if received.MonitorID != "api-prod" {
		t.Errorf("expected monitor api-prod, got %s", received.MonitorID)
	}
	if received.Status != string(StatusDown) {
		t.Errorf("expected status down, got %s", received.Status)
	}
}
