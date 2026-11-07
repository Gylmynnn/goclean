package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Dark gold theme (DiceSong-inspired)
var (
	// Base colors
	Background = lipgloss.Color("#0d1117")
	Surface    = lipgloss.Color("#161d27")
	SurfaceAlt = lipgloss.Color("#212a35")
	Foreground = lipgloss.Color("#c9ccd1")
	Comment    = lipgloss.Color("#5f6b76")
	Separator  = lipgloss.Color("#333b45")

	// Accent colors
	Red     = lipgloss.Color("#e06c75")
	Green   = lipgloss.Color("#9ecb7f")
	Yellow  = lipgloss.Color("#dfb95c")
	Blue    = lipgloss.Color("#61afef")
	Magenta = lipgloss.Color("#c678dd")
	Cyan    = lipgloss.Color("#58b7c6")
	Orange  = lipgloss.Color("#d98f5e")

	// Semantic colors
	Primary   = Yellow
	Secondary = Blue
	Danger    = Red
	Warning   = Orange
	Info      = Blue
	Muted     = Comment

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(Yellow).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Comment)

	// Badge chip styles (header title / path badges)
	BadgeStyle = lipgloss.NewStyle().
			Foreground(Yellow).
			Background(Surface).
			Bold(true).
			Padding(0, 1)

	CountStyle = lipgloss.NewStyle().
			Foreground(Yellow).
			Italic(true)

	PathBadgeStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Background(Surface).
			Padding(0, 1)

	// Table header style (gold column headers)
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(Yellow).
				Bold(true)

	// List item styles
	ListItemStyle = lipgloss.NewStyle().
			Foreground(Foreground)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(Yellow).
				Background(SurfaceAlt).
				Bold(true)

	CheckedItemStyle = lipgloss.NewStyle().
				Foreground(Yellow)

	// Separator rule style
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(Separator)

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
				Background(Yellow).
				Padding(0, 2).
				Bold(true)

	DangerButtonStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Red).
				Padding(0, 2).
				Bold(true)

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(Yellow)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(Comment)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Blue).
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
			Foreground(Blue)

	// Dialog styles
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Yellow).
			Padding(1, 2)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Yellow).
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
				BorderForeground(Yellow).
				Padding(1, 2)
)

// RenderSeparator renders a full-width horizontal rule
func RenderSeparator(width int) string {
	if width < 1 {
		return ""
	}
	return SeparatorStyle.Render(strings.Repeat("─", width))
}

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
