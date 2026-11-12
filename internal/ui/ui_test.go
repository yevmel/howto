package ui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestShowCommandPlain(t *testing.T) {
	var b bytes.Buffer
	newPlain(&b, false, 0).ShowCommand("ls -la |\nwc -l")
	if got, want := b.String(), "\n  ls -la |\n  wc -l\n\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestShowCommandBox(t *testing.T) {
	var b bytes.Buffer
	newPlain(&b, true, 80).ShowCommand("ls -la")
	want := "\n╭────────╮\n│ ls -la │\n╰────────╯\n\n"
	if got := b.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestShowCommandWrapsToTerminalWidth(t *testing.T) {
	var b bytes.Buffer
	cmd := "find . -type f -name '*.log' -size +100M -exec ls -lh {} + | sort -k5 -h"
	newPlain(&b, true, 30).ShowCommand(cmd)
	lines := strings.Split(strings.Trim(b.String(), "\n"), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected the command to wrap:\n%s", b.String())
	}
	for _, l := range lines {
		if w := lipgloss.Width(l); w > 30 {
			t.Errorf("line wider than the terminal (%d): %q", w, l)
		}
	}
	joined := strings.Join(strings.Fields(strings.ReplaceAll(b.String(), "│", "")), " ")
	if !strings.Contains(joined, "sort -k5 -h") {
		t.Errorf("command text lost while wrapping:\n%s", b.String())
	}
}

func TestErrorAndInfo(t *testing.T) {
	var b bytes.Buffer
	u := newPlain(&b, false, 0)
	u.Error(errors.New("boom"))
	u.Info("Not executed.")
	if got, want := b.String(), "howto: error: boom\nNot executed.\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSpinWithoutTerminalRunsAction(t *testing.T) {
	var b bytes.Buffer
	want := errors.New("failed")
	got := newPlain(&b, false, 0).Spin(context.Background(), "Thinking…", func(context.Context) error { return want })
	if !errors.Is(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if b.Len() != 0 {
		t.Errorf("unexpected output %q", b.String())
	}
}

func TestSpinOnTerminal(t *testing.T) {
	var b bytes.Buffer
	err := newPlain(&b, true, 80).Spin(context.Background(), "Thinking…", func(context.Context) error {
		time.Sleep(3 * spinnerInterval)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "Thinking…") || !strings.HasSuffix(out, "\r\x1b[2K\x1b[?25h") {
		t.Errorf("unexpected spinner output %q", out)
	}
}

func TestSpinReturnsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(2*spinnerInterval, cancel)
	block := make(chan struct{})
	defer close(block)

	start := time.Now()
	err := newPlain(&bytes.Buffer{}, true, 80).Spin(ctx, "x", func(context.Context) error {
		<-block // ignores ctx on purpose
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("Spin took %s after cancellation", d)
	}
}

func TestConfirmLine(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"YES\n", true},
		{" y \n", true},
		{"n\n", false},
		{"\n", false},
		{"yep\n", false},
		{"", false}, // EOF
	}
	for _, tt := range tests {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.WriteString(tt.input)
		_ = w.Close()

		var b bytes.Buffer
		got := newPlain(&b, false, 0).Confirm(context.Background(), r, "Execute? [y/N]")
		_ = r.Close()
		if got != tt.want {
			t.Errorf("input %q: got %v, want %v", tt.input, got, tt.want)
		}
		if !strings.HasPrefix(b.String(), "Execute? [y/N] ") {
			t.Errorf("prompt missing: %q", b.String())
		}
	}
}

func TestConfirmLineCancelled(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if newPlain(&bytes.Buffer{}, false, 0).Confirm(ctx, r, "?") {
		t.Error("cancelled confirmation must mean no")
	}
}
