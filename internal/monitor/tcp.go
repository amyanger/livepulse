package monitor

import (
	"context"
	"fmt"
	"net"
	"time"
)

type TCPCheck struct {
	id      string
	address string
	timeout time.Duration
}

func NewTCPCheck(id, address string, timeout time.Duration) *TCPCheck {
	return &TCPCheck{id: id, address: address, timeout: timeout}
}

func (c *TCPCheck) ID() string { return c.id }

func (c *TCPCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.address)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("tcp connect: %v", err)
		return result
	}
	conn.Close()

	result.Status = StatusHealthy
	return result
}
