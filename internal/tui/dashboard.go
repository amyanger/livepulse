package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/arjunmyanger/livepulse/internal/monitor"
	"github.com/arjunmyanger/livepulse/internal/tui/components"
)

// MonitorView holds the display state for a single monitor on the dashboard.
type MonitorView struct {
	ID        string
	Name      string
	Type      string
	Target    string
	Status    monitor.Status
	Latency   time.Duration
	LastCheck time.Time
	Error     string
	Sparkline *components.Sparkline
}

// statusIcon returns a colored status indicator for the given monitor status.
func statusIcon(s monitor.Status) string {
	switch s {
	case monitor.StatusHealthy:
		return healthyStyle.Render("●")
	case monitor.StatusDegraded:
		return degradedStyle.Render("●")
	case monitor.StatusDown:
		return downStyle.Render("●")
	default:
		return helpStyle.Render("●")
	}
}

// statusColor returns the color for a given status.
func statusColor(s monitor.Status) color.Color {
	switch s {
	case monitor.StatusHealthy:
		return colorHealthy
	case monitor.StatusDegraded:
		return colorDegraded
	case monitor.StatusDown:
		return colorDown
	default:
		return colorMuted
	}
}

// renderMonitorLine renders a single compact monitor row.
func renderMonitorLine(m MonitorView, width int) string {
	icon := statusIcon(m.Status)
	name := monitorNameStyle.Render(fmt.Sprintf("%-20s", m.Name))
	typeLabel := dimTypeStyle.Render(fmt.Sprintf("%-7s", m.Type))

	// Color latency based on status
	latStr := fmt.Sprintf("%dms", m.Latency.Milliseconds())
	var latency string
	if m.Status == monitor.StatusHealthy {
		latency = lipgloss.NewStyle().Foreground(colorDimFg).Render(fmt.Sprintf("%8s", latStr))
	} else {
		latency = lipgloss.NewStyle().Foreground(statusColor(m.Status)).Render(fmt.Sprintf("%8s", latStr))
	}

	spark := ""
	if m.Sparkline != nil {
		spark = m.Sparkline.ViewColored(statusColor(m.Status))
	}

	return fmt.Sprintf(" %s  %s %s  %s  %s", icon, name, typeLabel, latency, spark)
}

// renderExpandedRow renders the selected monitor with an expanded detail line.
func renderExpandedRow(m MonitorView, width int) string {
	icon := statusIcon(m.Status)
	name := lipgloss.NewStyle().Foreground(colorFg).Bold(true).Render(fmt.Sprintf("%-20s", m.Name))
	typeLabel := dimTypeStyle.Render(fmt.Sprintf("%-7s", m.Type))

	latStr := fmt.Sprintf("%dms", m.Latency.Milliseconds())
	latency := lipgloss.NewStyle().Foreground(colorFg).Bold(true).Render(fmt.Sprintf("%8s", latStr))

	spark := ""
	if m.Sparkline != nil {
		spark = m.Sparkline.ViewColored(statusColor(m.Status))
	}

	line1 := fmt.Sprintf("%s  %s %s  %s  %s", icon, name, typeLabel, latency, spark)

	// Detail line: target, last checked
	targetLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("target")
	targetVal := lipgloss.NewStyle().Foreground(colorDimFg).Render(truncate(m.Target, 30))
	checkedLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("checked")
	checkedVal := lipgloss.NewStyle().Foreground(colorDimFg).Render(timeAgo(m.LastCheck))

	line2 := fmt.Sprintf("     %s  %s    %s  %s", targetLabel, targetVal, checkedLabel, checkedVal)

	content := line1 + "\n" + line2
	return expandedRowStyle.Width(width - 2).Render(content)
}

// renderSeparator renders a subtle horizontal separator line.
func renderSeparator(width int) string {
	line := strings.Repeat("─", width-2)
	return separatorStyle.Render(" " + line)
}

// renderHelpBar renders the help bar with accent-colored keys.
func renderHelpBar(bindings []struct{ key, desc string }) string {
	var parts []string
	for _, b := range bindings {
		parts = append(parts, fmt.Sprintf("%s %s",
			helpKeyStyle.Render(b.key),
			helpDescStyle.Render(b.desc),
		))
	}
	return " " + strings.Join(parts, "  ")
}

// renderDashboard renders the main dashboard with all sections.
func renderDashboard(monitors []MonitorView, selected int, statusBar *components.StatusBar, eventLog *components.EventLog, eventsExpanded bool, width, height int) string {
	var sections []string

	// Gradient header
	rightInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("#BBBBFF")).
		Render(fmt.Sprintf("%d monitors · %s", len(monitors), time.Now().Format("15:04")))
	leftContent := "◆  livepulse"
	// Pad between left and right
	gap := width - len(leftContent) - lipgloss.Width(rightInfo) - 4
	if gap < 1 {
		gap = 1
	}
	headerContent := fmt.Sprintf("%s%*s%s", leftContent, gap, "", rightInfo)
	header := gradientHeaderStyle.Width(width).Render(headerContent)
	sections = append(sections, header)
	sections = append(sections, "")

	// Status bar pills
	if statusBar != nil {
		sections = append(sections, " "+statusBar.View())
		sections = append(sections, "")
	}

	// Monitor list
	if len(monitors) == 0 {
		empty := helpStyle.Render("  No monitors configured. Press 'a' to add one.")
		sections = append(sections, empty)
	} else {
		for i, m := range monitors {
			if i == selected {
				sections = append(sections, renderExpandedRow(m, width))
			} else {
				sections = append(sections, renderMonitorLine(m, width))
			}
			// Add separator between rows (but not after the last one)
			if i < len(monitors)-1 {
				sections = append(sections, renderSeparator(width))
			}
		}
	}
	sections = append(sections, "")

	// Event log — collapsed or expanded
	if eventsExpanded {
		sections = append(sections, eventLog.ViewExpanded(width))
	} else {
		sections = append(sections, eventLog.ViewCollapsed())
	}
	sections = append(sections, "")

	// Help bar with accent keys
	help := renderHelpBar([]struct{ key, desc string }{
		{"a", "add"},
		{"e", "edit"},
		{"d", "delete"},
		{"enter", "detail"},
		{"v", "events"},
		{"?", "help"},
		{"q", "quit"},
	})
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}

// timeAgo returns a human-readable "X ago" string.
func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}

// truncate shortens a string to max length, adding "…" if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
