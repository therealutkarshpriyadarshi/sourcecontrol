package ui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	// Primary colors
	ColorGreenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	ColorRedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	ColorYellowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	ColorBlueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	ColorCyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	ColorMagentaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Italic(true)

	// Status-specific styles
	ModifiedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Bold(true)
	DeletedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Bold(true)
	AddedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	UntrackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	// Layout styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5F5FFF")).
			PaddingTop(1).
			PaddingBottom(1).
			MarginBottom(1)

	InfoStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00BFFF")).
			PaddingTop(1).
			PaddingBottom(1)

	SectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Underline(true)

	CommitBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5F5FFF")).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)
)

// Icons
const (
	IconCheck     = "✓"
	IconModified  = "◉"
	IconDeleted   = "✗"
	IconAdded     = "+"
	IconUntracked = "?"
	IconBranch    = "⎇"
	IconCommit    = "⊚"
	IconAuthor    = "👤"
	IconDate      = "📅"
	IconSeparator = "│"
	IconCheckmark = "✓"
)

// Color wrapper functions
func Green(s string) string {
	return ColorGreenStyle.Render(s)
}

func Red(s string) string {
	return ColorRedStyle.Render(s)
}

func Yellow(s string) string {
	return ColorYellowStyle.Render(s)
}

func Blue(s string) string {
	return ColorBlueStyle.Render(s)
}

func Cyan(s string) string {
	return ColorCyanStyle.Render(s)
}

func Magenta(s string) string {
	return ColorMagentaStyle.Render(s)
}

// Layout rendering functions
func Header(text string) string {
	return HeaderStyle.Render(text)
}

func Section(text string) string {
	return SectionStyle.Render(text)
}

func Info(text string) string {
	return InfoStyle.Render(text)
}

func CommitBox(text string) string {
	return CommitBoxStyle.Render(text)
}
