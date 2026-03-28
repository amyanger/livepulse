package tui

import "charm.land/lipgloss/v2"

var (
	colorHealthy  = lipgloss.Color("#00FF00")
	colorDegraded = lipgloss.Color("#FFD700")
	colorDown     = lipgloss.Color("#FF0000")
	colorMuted    = lipgloss.Color("#666666")
	colorAccent   = lipgloss.Color("#7D56F4")
	colorBg       = lipgloss.Color("#1a1a2e")
	colorFg       = lipgloss.Color("#FAFAFA")

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	healthyStyle  = lipgloss.NewStyle().Foreground(colorHealthy).Bold(true)
	degradedStyle = lipgloss.NewStyle().Foreground(colorDegraded).Bold(true)
	downStyle     = lipgloss.NewStyle().Foreground(colorDown).Bold(true)

	helpStyle = lipgloss.NewStyle().Foreground(colorMuted)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333366")).
			Foreground(colorFg)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorFg).
			Background(colorAccent).
			Padding(0, 2).
			Width(80)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(16)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorFg)
)
