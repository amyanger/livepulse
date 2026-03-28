package monitor

import (
	"context"
	"time"
)

type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
)

type CheckResult struct {
	MonitorID string
	Status    Status
	Latency   time.Duration
	Error     string
	Timestamp time.Time
}

type Checker interface {
	Check(ctx context.Context) CheckResult
	ID() string
}
