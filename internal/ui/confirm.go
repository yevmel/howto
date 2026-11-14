package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Choice is the answer to the execution prompt.
type Choice int

const (
	Execute Choice = iota
	ExecuteSudo
	Nevermind
)

var choiceLabels = [...]string{
	Execute:     "Execute",
	ExecuteSudo: "Execute with sudo",
	Nevermind:   "Nevermind",
}

func (c Choice) String() string { return choiceLabels[c] }

// Choose asks how to run the command. On a terminal it shows a menu navigated with the arrow keys;
// otherwise a line is read ("y"/"yes" executes, "s"/"sudo" executes with sudo).
// Esc, Ctrl-C, EOF and read errors mean Nevermind.
func (u *UI) Choose(ctx context.Context, in *os.File, question string) Choice {
	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		fmt.Fprint(u.out, u.bold(question)+" [y/s/N] ")
		return u.chooseLine(ctx, in)
	}

	// Keys typed while waiting for the LLM must not choose for a command the user has not seen.
	_ = flushInput(fd)
	state, err := term.MakeRaw(fd)
	if err != nil {
		return Nevermind
	}
	defer func() { _ = term.Restore(fd, state) }()

	fmt.Fprint(u.out, "\x1b[?25l")
	defer fmt.Fprint(u.out, "\x1b[?25h")

	sel := Execute
	u.drawMenu(question, sel, false)
	buf := make([]byte, 32)
	for {
		// Read whole escape sequences at once, so a lone ESC is the Esc key.
		n, err := in.Read(buf)
		if err != nil {
			sel = Nevermind
			break
		}
		var done bool
		if sel, done = menuKey(sel, string(buf[:n])); done {
			break
		}
		u.drawMenu(question, sel, true)
	}

	// Collapse the menu into the question followed by the answer.
	fmt.Fprintf(u.out, "\r\x1b[%dA\x1b[J%s %s\r\n", len(choiceLabels), u.bold(question), sel)
	return sel
}

// menuKey applies a keypress to the selection; done reports that the selection is final.
func menuKey(sel Choice, key string) (Choice, bool) {
	switch key {
	case "\r", "\n":
		return sel, true
	case "\x1b", "\x03", "\x04": // Esc, Ctrl-C, Ctrl-D
		return Nevermind, true
	case "\x1b[A", "\x1bOA", "k", "\x10": // Up, Ctrl-P
		if sel > Execute {
			sel--
		}
	case "\x1b[B", "\x1bOB", "j", "\x0e", "\t": // Down, Ctrl-N
		if sel < Nevermind {
			sel++
		}
	}
	return sel, false
}

// drawMenu prints the question and options; redraw first moves back over the previous drawing.
// Lines end in \r\n because the terminal is in raw mode.
func (u *UI) drawMenu(question string, sel Choice, redraw bool) {
	var b strings.Builder
	if redraw {
		fmt.Fprintf(&b, "\r\x1b[%dA", len(choiceLabels))
	}
	b.WriteString("\x1b[2K" + u.bold(question))
	for _, line := range u.menuLines(sel) {
		b.WriteString("\r\n\x1b[2K" + line)
	}
	fmt.Fprint(u.out, b.String())
}

// menuLines renders the options, marking the selected one.
func (u *UI) menuLines(sel Choice) []string {
	hint := u.r.NewStyle().Foreground(muted)
	lines := make([]string, len(choiceLabels))
	for i, label := range choiceLabels {
		c := Choice(i)
		line := "  " + label
		if c == sel {
			line = u.r.NewStyle().Foreground(accent).Bold(true).Render("❯ "+label) + " " + hint.Render("(Enter)")
		} else if c == Nevermind {
			line += " " + hint.Render("(Esc)")
		}
		lines[i] = line
	}
	return lines
}

func (u *UI) bold(s string) string { return u.r.NewStyle().Bold(true).Render(s) }

func (u *UI) chooseLine(ctx context.Context, in *os.File) Choice {
	answer := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(in).ReadString('\n')
		answer <- line
	}()
	select {
	case line := <-answer:
		// Piped input is not echoed by a terminal; echo it to end the prompt line.
		a := strings.TrimSpace(line)
		fmt.Fprintln(u.out, a)
		switch strings.ToLower(a) {
		case "y", "yes":
			return Execute
		case "s", "sudo":
			return ExecuteSudo
		}
		return Nevermind
	case <-ctx.Done():
		fmt.Fprintln(u.out)
		return Nevermind
	}
}
