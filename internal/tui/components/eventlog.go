package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// Event represents a single event entry in the log.
type Event struct {
	Time    time.Time
	Monitor string
	Message string
	IsError bool
}

// EventLog maintains a rolling list of recent events for display.
// It supports a collapsed mode showing only a summary line.
type EventLog struct {
	events   []Event
	maxLines int
}

// NewEventLog creates an event log that retains up to maxLines entries.
func NewEventLog(maxLines int) *EventLog {
	return &EventLog{maxLines: maxLines}
}

// Add appends an event to the log, dropping the oldest if at capacity.
func (e *EventLog) Add(monitor, message string, isError bool) {
	e.events = append(e.events, Event{
		Time:    time.Now(),
		Monitor: monitor,
		Message: message,
		IsError: isError,
	})
	if len(e.events) > e.maxLines {
		e.events = e.events[len(e.events)-e.maxLines:]
	}
}

// recentCount returns how many events occurred in the last duration.
func (e *EventLog) recentCount(d time.Duration) int {
	cutoff := time.Now().Add(-d)
	count := 0
	for _, ev := range e.events {
		if ev.Time.After(cutoff) {
			count++
		}
	}
	return count
}

// ViewCollapsed renders a single summary line for the collapsed state.
func (e *EventLog) ViewCollapsed() string {
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B7DFF")).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	recent := e.recentCount(5 * time.Minute)
	summary := "no recent events"
	if recent > 0 {
		summary = fmt.Sprintf("%d new in last 5m", recent)
	}

	return fmt.Sprintf(" %s  %s",
		accentStyle.Render("▸ Events"),
		mutedStyle.Render(summary),
	)
}

// ViewExpanded renders the full event log with status icons.
func (e *EventLog) ViewExpanded(width int) string {
	accentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9B7DFF")).
		Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	// Header line
	headerLeft := accentStyle.Render("▾ RECENT EVENTS")
	showing := min(len(e.events), e.maxLines)
	headerRight := dimStyle.Render(fmt.Sprintf("showing %d of %d", showing, len(e.events)))

	// Pad between left and right
	headerGap := width - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 6
	if headerGap < 1 {
		headerGap = 1
	}
	header := fmt.Sprintf(" %s%*s%s", headerLeft, headerGap, "", headerRight)

	if len(e.events) == 0 {
		noEvents := mutedStyle.Render("  No events yet...")
		return fmt.Sprintf("%s\n%s", header, noEvents)
	}

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ADE80"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	result := header + "\n"

	// Border top
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	borderWidth := width - 5
	if borderWidth < 1 {
		borderWidth = 1
	}
	border := borderStyle.Render(" ╭" + strings.Repeat("─", borderWidth) + "╮")
	result += border + "\n"

	start := 0
	if len(e.events) > e.maxLines {
		start = len(e.events) - e.maxLines
	}

	for _, ev := range e.events[start:] {
		ts := timeStyle.Render(ev.Time.Format("15:04:05"))

		// Status icon based on event type
		icon := okStyle.Render("▲") // default: recovery
		if ev.IsError {
			icon = errStyle.Render("▼")
		} else if strings.Contains(ev.Message, "degraded") || strings.Contains(ev.Message, "threshold") || strings.Contains(ev.Message, "slow") {
			icon = warnStyle.Render("⚠")
		}

		line := fmt.Sprintf(" │ %s  %s  %s  %s",
			ts, icon, nameStyle.Render(ev.Monitor), mutedStyle.Render(ev.Message))
		result += line + "\n"
	}

	// Border bottom
	borderBottom := borderStyle.Render(" ╰" + strings.Repeat("─", borderWidth) + "╯")
	result += borderBottom

	return result
}

// View renders the event log (expanded mode, for backward compatibility).
func (e *EventLog) View() string {
	return e.ViewExpanded(80)
}

// Clear removes all events from the log.
func (e *EventLog) Clear() {
	e.events = nil
}
