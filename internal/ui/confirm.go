package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

// Confirm asks a yes/no question; only "y" means yes. On a terminal a single key answers,
// otherwise a line is read ("y" or "yes"). Ctrl-C, EOF and read errors mean no.
func (u *UI) Confirm(ctx context.Context, in *os.File, question string) bool {
	fmt.Fprint(u.out, u.r.NewStyle().Bold(true).Render(question)+" ")

	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		return u.confirmLine(ctx, in)
	}

	key, err := readKey(in)
	if err != nil {
		key = ""
	}
	r, _ := utf8.DecodeRuneInString(key)
	if key != "" && unicode.IsPrint(r) {
		fmt.Fprintln(u.out, string(r))
	} else {
		fmt.Fprintln(u.out)
	}
	return r == 'y' || r == 'Y'
}

// readKey reads one keypress without waiting for Enter, after discarding typed-ahead input.
func readKey(in *os.File) (string, error) {
	fd := int(in.Fd())
	// A "y" typed while waiting for the LLM must not confirm a command the user has not seen.
	_ = flushInput(fd)

	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, state) }()

	// Read up to 32 bytes so multi-byte keys (arrows, umlauts) are consumed whole.
	buf := make([]byte, 32)
	n, err := in.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

func (u *UI) confirmLine(ctx context.Context, in *os.File) bool {
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
		a = strings.ToLower(a)
		return a == "y" || a == "yes"
	case <-ctx.Done():
		fmt.Fprintln(u.out)
		return false
	}
}
