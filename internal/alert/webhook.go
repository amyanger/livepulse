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
