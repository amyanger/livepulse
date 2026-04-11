package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

type ProcessMetrics struct {
	PID        int32
	CPUPercent float64
	MemPercent float64
	MemMB      float64
}

type ProcessCheck struct {
	id           string
	name         string
	cpuThreshold float64
	memThreshold float64
}

func NewProcessCheck(id, processName string, cpuThreshold, memThreshold float64) *ProcessCheck {
	return &ProcessCheck{
		id:           id,
		name:         processName,
		cpuThreshold: cpuThreshold,
		memThreshold: memThreshold,
	}
}

func (c *ProcessCheck) ID() string { return c.id }

func (c *ProcessCheck) Collect(ctx context.Context) (ProcessMetrics, error) {
	var m ProcessMetrics

	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return m, fmt.Errorf("list processes: %w", err)
	}

	// Find matching processes by name
	var candidates []*process.Process
	for _, p := range procs {
		name, err := p.NameWithContext(ctx)
		if err != nil {
			continue
		}
		if strings.EqualFold(name, c.name) {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return m, fmt.Errorf("process not found: %s", c.name)
	}

	// Pick the process with highest CPU usage
	var best *process.Process
	var bestCPU float64
	for _, p := range candidates {
		cpu, err := p.CPUPercentWithContext(ctx)
		if err != nil {
			continue
		}
		if best == nil || cpu > bestCPU {
			best = p
			bestCPU = cpu
		}
	}

	if best == nil {
		best = candidates[0]
	}

	m.PID = best.Pid

	cpuPct, err := best.CPUPercentWithContext(ctx)
	if err == nil {
		m.CPUPercent = cpuPct
	}

	memInfo, err := best.MemoryInfoWithContext(ctx)
	if err == nil && memInfo != nil {
		m.MemMB = float64(memInfo.RSS) / (1024 * 1024)
	}

	memPct, err := best.MemoryPercentWithContext(ctx)
	if err == nil {
		m.MemPercent = float64(memPct)
	}

	return m, nil
}

func (c *ProcessCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	metrics, err := c.Collect(ctx)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = err.Error()
		return result
	}

	if metrics.CPUPercent > c.cpuThreshold || metrics.MemPercent > c.memThreshold {
		result.Status = StatusDegraded
		result.Error = fmt.Sprintf("high usage — cpu:%.0f%% mem:%.0f%% (pid:%d)",
			metrics.CPUPercent, metrics.MemPercent, metrics.PID)
		return result
	}

	result.Status = StatusHealthy
	return result
}
