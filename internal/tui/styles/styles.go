package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Everblush Theme Colors
var (
	// Base colors
	Background = lipgloss.Color("#141b1e")
	Surface    = lipgloss.Color("#1e2528")
	SurfaceAlt = lipgloss.Color("#282e31")
	Foreground = lipgloss.Color("#dadada")
	Comment    = lipgloss.Color("#6d7579")

	// Accent colors
	Red     = lipgloss.Color("#e57474")
	Green   = lipgloss.Color("#8ccf7e")
	Yellow  = lipgloss.Color("#e5c76b")
	Blue    = lipgloss.Color("#67b0e8")
	Magenta = lipgloss.Color("#c47fd5")
	Cyan    = lipgloss.Color("#6cbfbf")
	Orange  = lipgloss.Color("#ef7e57")

	// Semantic colors
	Primary   = Cyan
	Secondary = Green
	Danger    = Red
	Warning   = Yellow
	Info      = Blue
	Muted     = Comment

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Comment)

	// List item styles
	ListItemStyle = lipgloss.NewStyle().
			Foreground(Foreground)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(Cyan).
				Bold(true)

	CheckedItemStyle = lipgloss.NewStyle().
				Foreground(Green)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Orange)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Foreground(Foreground).
			Background(Surface).
			Padding(0, 2)

	ActiveButtonStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Cyan).
				Padding(0, 2).
				Bold(true)

	DangerButtonStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Red).
				Padding(0, 2).
				Bold(true)

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(Cyan)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(Comment)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Magenta).
			Bold(true)

	// Size indicator styles
	SizeSmallStyle = lipgloss.NewStyle().
			Foreground(Green)

	SizeMediumStyle = lipgloss.NewStyle().
			Foreground(Yellow)

	SizeLargeStyle = lipgloss.NewStyle().
			Foreground(Red)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Cyan)

	// Dialog styles
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Orange).
			Padding(1, 2)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Magenta).
			Bold(true)

	// TextMuted style
	TextMutedStyle = lipgloss.NewStyle().
			Foreground(Comment)

	// Box style for desktop
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SurfaceAlt).
			Padding(1, 2)

	// Selected box style
	SelectedBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Cyan).
				Padding(1, 2)
)

// Layouting responsive breakpoints
const (
	MobileWidth  = 60
	TabletWidth  = 100
	DesktopWidth = 120
)

// IsMobile returns true if width is mobile size
func IsMobile(width int) bool {
	return width < MobileWidth
}

// IsDesktop returns true if width is desktop size
func IsDesktop(width int) bool {
	return width >= TabletWidth
}

// GetSizeStyle returns appropriate style based on size
func GetSizeStyle(bytes int64) lipgloss.Style {
	switch {
	case bytes < 1024*1024:
		return SizeSmallStyle
	case bytes < 100*1024*1024:
		return SizeMediumStyle
	default:
		return SizeLargeStyle
	}
}

// GetContentWidth returns the content width based on terminal width
func GetContentWidth(termWidth int) int {
	if termWidth < MobileWidth {
		return termWidth - 2
	}
	if termWidth < TabletWidth {
		return termWidth - 4
	}
	maxWidth := 90
	if termWidth > maxWidth+10 {
		return maxWidth
	}
	return termWidth - 10
}

// CenterHorizontally centers content horizontally
func CenterHorizontally(content string, termWidth, contentWidth int) string {
	if termWidth <= contentWidth {
		return content
	}
	padding := (termWidth - contentWidth) / 2
	paddingStr := strings.Repeat(" ", padding)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = paddingStr + line
	}
	return strings.Join(lines, "\n")
}

func CenterVertically(content string, termHeight int) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)

	if termHeight <= contentHeight {
		return content
	}

	padding := (termHeight - contentHeight) / 2
	topPadding := strings.Repeat("\n", padding)
	return topPadding + content
}

// CenterBoth centers content both horizontally and vertically
func CenterBoth(content string, termWidth, termHeight, contentWidth int) string {
	centered := CenterHorizontally(content, termWidth, contentWidth)
	return CenterVertically(centered, termHeight)
}
