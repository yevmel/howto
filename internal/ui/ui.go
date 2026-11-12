// Package ui renders howto's terminal output: spinner, command box, prompt and messages.
// All output goes to one writer (stderr), so stdout belongs to the executed command.
package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

var (
	accent = lipgloss.AdaptiveColor{Light: "#5A3FC0", Dark: "#B4A3FF"}
	muted  = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#6C6C6C"}
	danger = lipgloss.AdaptiveColor{Light: "#C62828", Dark: "#FF6B6B"}
)

// UI writes to one output, styled when it is a terminal.
type UI struct {
	out   io.Writer
	tty   bool // out is a terminal: spinner and box are shown
	width int  // terminal width; 0 if unknown
	r     *lipgloss.Renderer
}

// New returns a UI writing to out; colors are off without a terminal or with NO_COLOR set.
func New(out *os.File, getenv func(string) string) *UI {
	fd := int(out.Fd())
	tty := term.IsTerminal(fd)
	r := lipgloss.NewRenderer(out)
	if !tty || getenv("NO_COLOR") != "" {
		r.SetColorProfile(termenv.Ascii)
	} else {
		// Query the background now, not while the spinner draws.
		r.HasDarkBackground()
	}
	width := 0
	if tty {
		width, _, _ = term.GetSize(fd)
	}
	return &UI{out: out, tty: tty, width: width, r: r}
}

// newPlain returns a UI without terminal features, for tests.
func newPlain(out io.Writer, tty bool, width int) *UI {
	r := lipgloss.NewRenderer(out)
	r.SetColorProfile(termenv.Ascii)
	return &UI{out: out, tty: tty, width: width, r: r}
}

func (u *UI) colored() bool { return u.r.ColorProfile() != termenv.Ascii }

// Error prints "howto: error: <err>".
func (u *UI) Error(err error) {
	label := u.r.NewStyle().Foreground(danger).Bold(true).Render("howto: error:")
	fmt.Fprintf(u.out, "%s %v\n", label, err)
}

// Info prints a dimmed status message.
func (u *UI) Info(msg string) {
	fmt.Fprintln(u.out, u.r.NewStyle().Foreground(muted).Render(msg))
}
