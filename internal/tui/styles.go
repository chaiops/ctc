package tui

import "github.com/charmbracelet/lipgloss"

// Palette — a calm, high-contrast terminal theme.
var (
	colorAccent  = lipgloss.Color("#7D56F4") // violet
	colorAccent2 = lipgloss.Color("#43BF6D") // green
	colorMuted   = lipgloss.Color("#6C7086")
	colorText    = lipgloss.Color("#CDD6F4")
	colorBg      = lipgloss.Color("#1E1E2E")
	colorWarn    = lipgloss.Color("#F38BA8")
	colorYellow  = lipgloss.Color("#F9E2AF")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorAccent).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	// List rows.
	cursorStyle   = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	checkedStyle  = lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)
	rowStyle      = lipgloss.NewStyle().Foreground(colorText)
	rowSelStyle   = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	metaStyle     = lipgloss.NewStyle().Foreground(colorMuted)
	imageStyle    = lipgloss.NewStyle().Foreground(colorYellow)
	runningBadge  = lipgloss.NewStyle().Foreground(colorAccent2)
	stoppedBadge  = lipgloss.NewStyle().Foreground(colorMuted)
	selectedRowBg = lipgloss.NewStyle().Background(lipgloss.Color("#313244"))

	// Footer help.
	helpStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	helpKeyStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	// Preview.
	previewFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	statusOKStyle   = lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)
	statusErrStyle  = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	statusWarnStyle = lipgloss.NewStyle().Foreground(colorYellow)

	// YAML syntax highlighting.
	yamlKeyStyle     = lipgloss.NewStyle().Foreground(colorAccent)
	yamlValueStyle   = lipgloss.NewStyle().Foreground(colorText)
	yamlListStyle    = lipgloss.NewStyle().Foreground(colorAccent2)
	yamlCommentStyle = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
)
