package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemMetrics struct {
	CPUPercent  float64
	MemPercent  float64
	MemUsedGB   float64
	MemTotalGB  float64
	DiskPercent float64
	DiskUsedGB  float64
	DiskTotalGB float64
}

type SystemCheck struct {
	id string
}

func NewSystemCheck(id string) *SystemCheck {
	return &SystemCheck{id: id}
}

func (c *SystemCheck) ID() string { return c.id }

func (c *SystemCheck) Collect() (SystemMetrics, error) {
	var m SystemMetrics

	cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return m, fmt.Errorf("cpu: %w", err)
	}
	if len(cpuPercent) > 0 {
		m.CPUPercent = cpuPercent[0]
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return m, fmt.Errorf("memory: %w", err)
	}
	m.MemPercent = memInfo.UsedPercent
	m.MemUsedGB = float64(memInfo.Used) / (1024 * 1024 * 1024)
	m.MemTotalGB = float64(memInfo.Total) / (1024 * 1024 * 1024)

	diskInfo, err := disk.Usage("/")
	if err != nil {
		return m, fmt.Errorf("disk: %w", err)
	}
	m.DiskPercent = diskInfo.UsedPercent
	m.DiskUsedGB = float64(diskInfo.Used) / (1024 * 1024 * 1024)
	m.DiskTotalGB = float64(diskInfo.Total) / (1024 * 1024 * 1024)

	return m, nil
}

func (c *SystemCheck) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		MonitorID: c.id,
		Timestamp: start,
	}

	metrics, err := c.Collect()
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("collect metrics: %v", err)
		return result
	}

	// Degraded if any metric > 90%
	if metrics.CPUPercent > 90 || metrics.MemPercent > 90 || metrics.DiskPercent > 90 {
		result.Status = StatusDegraded
		result.Error = fmt.Sprintf("high usage — cpu:%.0f%% mem:%.0f%% disk:%.0f%%",
			metrics.CPUPercent, metrics.MemPercent, metrics.DiskPercent)
		return result
	}

	result.Status = StatusHealthy
	return result
}
