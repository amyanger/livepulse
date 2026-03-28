package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/arjunmyanger/livepulse/internal/monitor"
	"github.com/arjunmyanger/livepulse/internal/tui/components"
)

// renderDetail renders the single monitor detail view with status, type, target,
// latency, last check time, error, latency sparkline history, and recent check results.
func renderDetail(m MonitorView, results []monitor.CheckResult, sparkline *components.Sparkline, width int) string {
	var sections []string

	// Header with monitor name
	header := headerStyle.Width(width).Render(fmt.Sprintf("Monitor: %s", m.Name))
	sections = append(sections, header)
	sections = append(sections, "")

	// Status/Type/Target/Latency/Last Check/Error fields
	fields := []struct {
		label string
		value string
	}{
		{"Status", string(m.Status)},
		{"Type", m.Type},
		{"Target", m.Target},
		{"Latency", fmt.Sprintf("%dms", m.Latency.Milliseconds())},
		{"Last Check", formatTime(m.LastCheck)},
	}

	for _, f := range fields {
		line := fmt.Sprintf("  %s %s", labelStyle.Render(f.label), renderFieldValue(f.label, f.value, m.Status))
		sections = append(sections, line)
	}

	if m.Error != "" {
		errLine := fmt.Sprintf("  %s %s", labelStyle.Render("Error"), downStyle.Render(m.Error))
		sections = append(sections, errLine)
	}
	sections = append(sections, "")

	// Latency History sparkline
	sparkTitle := titleStyle.Render("Latency History")
	sections = append(sections, sparkTitle)
	if sparkline != nil {
		sparkBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1).
			Width(width - 4)
		sections = append(sections, sparkBorder.Render(sparkline.View()))
	}
	sections = append(sections, "")

	// Recent Checks list (last 10)
	checksTitle := titleStyle.Render("Recent Checks")
	sections = append(sections, checksTitle)

	if len(results) == 0 {
		sections = append(sections, helpStyle.Render("  No check results yet."))
	} else {
		// Show last 10 results, most recent first
		start := 0
		if len(results) > 10 {
			start = len(results) - 10
		}
		for i := len(results) - 1; i >= start; i-- {
			r := results[i]
			icon := statusIcon(r.Status)
			ts := helpStyle.Render(r.Timestamp.Format("15:04:05"))
			lat := fmt.Sprintf("%dms", r.Latency.Milliseconds())

			line := fmt.Sprintf("  %s  %s  %8s", icon, ts, lat)
			if r.Error != "" {
				line += "  " + downStyle.Render(r.Error)
			}
			sections = append(sections, line)
		}
	}
	sections = append(sections, "")

	// Help bar
	help := helpStyle.Render("  esc:back  e:edit  d:delete")
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}

// formatTime formats a time for display, showing "never" for zero values.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("2006-01-02 15:04:05")
}

// renderFieldValue applies status-appropriate styling to the Status field value;
// other fields use the default value style.
func renderFieldValue(label, value string, status monitor.Status) string {
	if label == "Status" {
		switch status {
		case monitor.StatusHealthy:
			return healthyStyle.Render(value)
		case monitor.StatusDegraded:
			return degradedStyle.Render(value)
		case monitor.StatusDown:
			return downStyle.Render(value)
		}
	}
	return valueStyle.Render(value)
}
