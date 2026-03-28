package tui

import (
	"fmt"
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

// renderMonitorLine renders a single monitor row with status icon, name, type label,
// latency, and sparkline. The selected row uses selectedStyle.
func renderMonitorLine(m MonitorView, selected bool, width int) string {
	icon := statusIcon(m.Status)
	typeLabel := helpStyle.Render(fmt.Sprintf("[%s]", m.Type))
	latency := fmt.Sprintf("%dms", m.Latency.Milliseconds())

	spark := ""
	if m.Sparkline != nil {
		spark = m.Sparkline.View()
	}

	// Build the line content: icon  name  [type]  latency  sparkline
	line := fmt.Sprintf(" %s  %-20s %s  %8s  %s", icon, m.Name, typeLabel, latency, spark)

	if selected {
		// Pad to fill width for selected highlight
		padded := line
		if len(padded) < width {
			padded += strings.Repeat(" ", width-len(padded))
		}
		return selectedStyle.Render(padded)
	}

	return line
}

// renderDashboard renders the main dashboard with monitor list, sparklines,
// status bar, event log, and help bar.
func renderDashboard(monitors []MonitorView, selected int, statusBar *components.StatusBar, eventLog *components.EventLog, width, height int) string {
	var sections []string

	// Header
	header := headerStyle.Width(width).Render("livepulse -- Real-Time Infrastructure Monitor")
	sections = append(sections, header)
	sections = append(sections, "")

	// Status bar summary
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
			sections = append(sections, renderMonitorLine(m, i == selected, width))
		}
	}
	sections = append(sections, "")

	// Event log panel
	eventTitle := titleStyle.Render("Events")
	eventBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Width(width - 4)

	eventContent := ""
	if eventLog != nil {
		eventContent = eventLog.View()
	}
	sections = append(sections, eventTitle)
	sections = append(sections, eventBorder.Render(eventContent))
	sections = append(sections, "")

	// Help bar
	help := helpStyle.Render("  a:add  e:edit  d:delete  enter:detail  ?:help  q:quit")
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}
