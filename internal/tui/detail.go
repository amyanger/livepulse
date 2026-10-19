package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/amyanger/livepulse/internal/monitor"
	"github.com/amyanger/livepulse/internal/tui/components"
)

// renderDetail renders the single monitor detail view with polished styling.
func renderDetail(m MonitorView, results []monitor.CheckResult, sparkline *components.Sparkline, width int) string {
	var sections []string

	// Gradient header with monitor name
	header := gradientHeaderStyle.Width(width).Render(fmt.Sprintf("◆  %s", m.Name))
	sections = append(sections, header)
	sections = append(sections, "")

	// Status as pill badge
	statusPill := renderStatusPill(m.Status)
	sections = append(sections, "  "+statusPill)
	sections = append(sections, "")

	// Info fields
	fields := []struct {
		label string
		value string
	}{
		{"Type", m.Type},
		{"Target", m.Target},
		{"Latency", fmt.Sprintf("%dms", m.Latency.Milliseconds())},
		{"Last Check", formatTime(m.LastCheck)},
	}

	for _, f := range fields {
		line := fmt.Sprintf("  %s %s", labelStyle.Render(f.label), valueStyle.Render(f.value))
		sections = append(sections, line)
	}

	if m.Error != "" {
		errLine := fmt.Sprintf("  %s %s", labelStyle.Render("Error"), errorBgStyle.Render(" "+m.Error+" "))
		sections = append(sections, errLine)
	}
	sections = append(sections, "")

	// Latency History sparkline
	sparkHeader := sectionHeaderStyle.Render("  LATENCY HISTORY")
	sections = append(sections, sparkHeader)
	if sparkline != nil {
		sparkBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1).
			Width(width - 4)
		sections = append(sections, sparkBorder.Render(sparkline.ViewColored(statusColor(m.Status))))
	}
	sections = append(sections, "")

	// Recent Checks list (last 10)
	checksHeader := sectionHeaderStyle.Render("  RECENT CHECKS")
	sections = append(sections, checksHeader)

	if len(results) == 0 {
		sections = append(sections, helpStyle.Render("  No check results yet."))
	} else {
		start := 0
		if len(results) > 10 {
			start = len(results) - 10
		}
		for idx, i := 0, len(results)-1; i >= start; i, idx = i-1, idx+1 {
			r := results[i]
			icon := statusIcon(r.Status)
			ts := helpStyle.Render(r.Timestamp.Format("15:04:05"))
			lat := fmt.Sprintf("%dms", r.Latency.Milliseconds())

			line := fmt.Sprintf("  %s  %s  %8s", icon, ts, lat)
			if r.Error != "" {
				line += "  " + errorBgStyle.Render(" "+r.Error+" ")
			}

			// Alternating row tinting (odd rows get subtle bg)
			if idx%2 == 1 {
				line = altRowStyle.Render(line)
			}

			sections = append(sections, line)
		}
	}
	sections = append(sections, "")

	// Help bar with accent keys
	help := renderHelpBar([]struct{ key, desc string }{
		{"esc", "back"},
		{"e", "edit"},
		{"d", "delete"},
	})
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}

// renderStatusPill renders a status value as a colored pill badge.
func renderStatusPill(s monitor.Status) string {
	var fg, bg string
	switch s {
	case monitor.StatusHealthy:
		fg = "#4ADE80"
		bg = "#1a2e1a"
	case monitor.StatusDegraded:
		fg = "#FBBF24"
		bg = "#2e2a1a"
	case monitor.StatusDown:
		fg = "#F87171"
		bg = "#2e1a1a"
	default:
		fg = "#888888"
		bg = "#1a1a1a"
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("● %s", string(s)))
}

// formatTime formats a time for display, showing "never" for zero values.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("2006-01-02 15:04:05")
}
