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
