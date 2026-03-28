package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type CustomCheck struct {
	id      string
	command string
	timeout time.Duration
}

func NewCustomCheck(id, command string, timeout time.Duration) *CustomCheck {
	return &CustomCheck{id: id, command: command, timeout: timeout}
}

func (c *CustomCheck) ID() string { return c.id }

func (c *CustomCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", c.command)
	output, err := cmd.CombinedOutput()
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("command failed: %v (output: %s)", err, string(output))
		return result
	}

	result.Status = StatusHealthy
	return result
}
