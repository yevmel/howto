package ui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ShowCommand prints the suggested command: highlighted in a box on a terminal, indented otherwise.
func (u *UI) ShowCommand(command string) {
	if !u.tty {
		fmt.Fprintf(u.out, "\n  %s\n\n", strings.ReplaceAll(command, "\n", "\n  "))
		return
	}

	style := u.r.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 1)
	// Wrap commands wider than the terminal; Width excludes the border.
	if u.width > 0 && lipgloss.Width(command)+style.GetHorizontalFrameSize() > u.width {
		style = style.Width(u.width - style.GetHorizontalBorderSize())
	}
	fmt.Fprintf(u.out, "\n%s\n\n", style.Render(u.highlight(command)))
}

// highlight colors the command as shell code for the terminal's color profile.
func (u *UI) highlight(command string) string {
	var formatter string
	switch u.r.ColorProfile() {
	case termenv.TrueColor:
		formatter = "terminal16m"
	case termenv.ANSI256:
		formatter = "terminal256"
	case termenv.ANSI:
		formatter = "terminal16"
	default:
		return command
	}
	theme := "monokailight"
	if u.r.HasDarkBackground() {
		theme = "monokai"
	}

	var b strings.Builder
	if err := quick.Highlight(&b, command, "bash", formatter, theme); err != nil {
		return command
	}
	return strings.TrimRight(b.String(), "\n")
}
