package tui

import "charm.land/lipgloss/v2"

var (
	// Status colors — softened palette
	colorHealthy  = lipgloss.Color("#4ADE80")
	colorDegraded = lipgloss.Color("#FBBF24")
	colorDown     = lipgloss.Color("#F87171")

	// UI colors
	colorMuted      = lipgloss.Color("#888888")
	colorAccent     = lipgloss.Color("#7D56F4")
	colorAccentLight = lipgloss.Color("#9B7DFF")
	colorBg         = lipgloss.Color("#1a1a2e")
	colorFg         = lipgloss.Color("#FAFAFA")
	colorDimFg      = lipgloss.Color("#AAAAAA")
	colorSeparator  = lipgloss.Color("#2a2a4e")

	// Status text styles
	healthyStyle  = lipgloss.NewStyle().Foreground(colorHealthy).Bold(true)
	degradedStyle = lipgloss.NewStyle().Foreground(colorDegraded).Bold(true)
	downStyle     = lipgloss.NewStyle().Foreground(colorDown).Bold(true)

	// Layout styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(colorAccentLight).
				Bold(true)

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

	// New styles for visual polish

	gradientHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorFg).
				Background(colorAccent).
				Padding(0, 2)

	expandedRowStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(colorAccent).
				Background(lipgloss.Color("#252545")).
				Padding(0, 1)

	expandedDetailStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	separatorStyle = lipgloss.NewStyle().
			Foreground(colorSeparator)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	monitorNameStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Bold(true)

	dimTypeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	errorBgStyle = lipgloss.NewStyle().
			Foreground(colorDown).
			Background(lipgloss.Color("#2a1a1a"))

	altRowStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e38"))
)
