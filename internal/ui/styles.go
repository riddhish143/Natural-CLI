package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#00D7FF") // Cyan
	successColor   = lipgloss.Color("#00FF87") // Green
	warningColor   = lipgloss.Color("#FFD700") // Yellow
	errorColor     = lipgloss.Color("#FF5F5F") // Red
	infoColor      = lipgloss.Color("#87AFFF") // Blue
	accentColor    = lipgloss.Color("#FF87FF") // Magenta
	subtleColor    = lipgloss.Color("#6C6C6C") // Gray
	textColor      = lipgloss.Color("#FAFAFA") // White

	// Base styles
	boxWidth = getTerminalWidth()

	// Answer box style - rounded borders with cyan
	answerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Margin(1, 2).
			Width(boxWidth)

	// Success box style
	successStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(successColor).
			Padding(1, 2).
			Margin(1, 2).
			Width(boxWidth)

	// Error box style
	errorStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(errorColor).
			Padding(1, 2).
			Margin(1, 2).
			Width(boxWidth)

	// Warning box style
	warningStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(warningColor).
			Padding(1, 2).
			Margin(1, 2).
			Width(boxWidth)

	// Info box style
	infoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(infoColor).
			Padding(1, 2).
			Margin(1, 2).
			Width(boxWidth)

	// Command box style - rounded border for consistency
	commandStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Margin(1, 2).
			Width(boxWidth)

	// Blocked command style
	blockedStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(errorColor).
			Padding(1, 2).
			Margin(1, 2).
			Width(boxWidth)

	// Output box style - for command execution output
	outputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtleColor).
			Padding(1, 2).
			Margin(0, 2).
			Width(boxWidth)

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	successTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(successColor).
				MarginBottom(1)

	errorTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(errorColor).
			MarginBottom(1)

	warningTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warningColor).
				MarginBottom(1)

	// Content styles
	commandTextStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(textColor)

	explanationStyle = lipgloss.NewStyle().
				Foreground(subtleColor).
				Italic(true)

	// Icon styles
	successIcon = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			Render("✓")

	errorIcon = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Render("✗")

	warningIcon = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true).
			Render("⚠")

	infoIcon = lipgloss.NewStyle().
			Foreground(infoColor).
			Bold(true).
			Render("ℹ")

	commandIcon = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render("❯")

	blockedIcon = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Render("⛔")

	// Prompt style
	promptStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	promptArrowStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)
)

func getTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 70
	}
	// Leave margin and cap width
	usable := w - 6
	if usable > 90 {
		usable = 90
	}
	if usable < 50 {
		usable = 50
	}
	return usable
}

func refreshWidth() {
	boxWidth = getTerminalWidth()
	answerStyle = answerStyle.Width(boxWidth)
	successStyle = successStyle.Width(boxWidth)
	errorStyle = errorStyle.Width(boxWidth)
	warningStyle = warningStyle.Width(boxWidth)
	infoStyle = infoStyle.Width(boxWidth)
	commandStyle = commandStyle.Width(boxWidth)
	blockedStyle = blockedStyle.Width(boxWidth)
	outputStyle = outputStyle.Width(boxWidth)
}
