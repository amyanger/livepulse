package components

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// StatusBar displays a summary of monitor health counts.
type StatusBar struct {
	healthy  int
	degraded int
	down     int
	total    int
	width    int
}

// NewStatusBar creates a status bar with the given display width.
func NewStatusBar(width int) *StatusBar {
	return &StatusBar{width: width}
}

// Update refreshes the status bar counts.
func (s *StatusBar) Update(healthy, degraded, down int) {
	s.healthy = healthy
	s.degraded = degraded
	s.down = down
	s.total = healthy + degraded + down
}

// View renders the status bar as a styled string.
func (s *StatusBar) View() string {
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	return fmt.Sprintf("%s %s  %s %s  %s %s  %s %s",
		greenStyle.Render("●"), greenStyle.Render(fmt.Sprintf("%d healthy", s.healthy)),
		yellowStyle.Render("●"), yellowStyle.Render(fmt.Sprintf("%d degraded", s.degraded)),
		redStyle.Render("●"), redStyle.Render(fmt.Sprintf("%d down", s.down)),
		mutedStyle.Render("|"), mutedStyle.Render(fmt.Sprintf("%d total", s.total)),
	)
}
