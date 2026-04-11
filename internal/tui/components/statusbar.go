package components

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// StatusBar displays a summary of monitor health counts as pill badges.
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

// View renders the status bar as pill-shaped badges with tinted backgrounds.
func (s *StatusBar) View() string {
	healthyPill := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ADE80")).
		Background(lipgloss.Color("#1a2e1a")).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("● %d healthy", s.healthy))

	degradedPill := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24")).
		Background(lipgloss.Color("#2e2a1a")).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("● %d degraded", s.degraded))

	downPill := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F87171")).
		Background(lipgloss.Color("#2e1a1a")).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("● %d down", s.down))

	return fmt.Sprintf("%s  %s  %s", healthyPill, degradedPill, downPill)
}
