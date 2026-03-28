package components

import (
	"fmt"
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

// View renders the event log as a styled string.
func (e *EventLog) View() string {
	if len(e.events) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render("  No events yet...")
	}

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	var result string
	start := 0
	if len(e.events) > e.maxLines {
		start = len(e.events) - e.maxLines
	}

	for _, ev := range e.events[start:] {
		ts := timeStyle.Render(ev.Time.Format("15:04:05"))
		style := okStyle
		if ev.IsError {
			style = errStyle
		}
		line := fmt.Sprintf("  %s  %s  %s\n", ts, style.Render(ev.Monitor), ev.Message)
		result += line
	}

	return result
}

// Clear removes all events from the log.
func (e *EventLog) Clear() {
	e.events = nil
}
