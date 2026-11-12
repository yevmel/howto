package ui

import (
	"context"
	"fmt"
	"time"
)

var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

const spinnerInterval = 80 * time.Millisecond

// Spin runs action while showing a spinner with title; without a terminal it just runs action.
// It returns ctx.Err() as soon as ctx is cancelled (e.g. Ctrl-C), without waiting for action.
func (u *UI) Spin(ctx context.Context, title string, action func(context.Context) error) error {
	if !u.tty {
		return action(ctx)
	}

	done := make(chan error, 1)
	go func() { done <- action(ctx) }()

	frameStyle := u.r.NewStyle().Foreground(accent)
	title = u.r.NewStyle().Foreground(muted).Render(title)
	fmt.Fprint(u.out, "\x1b[?25l")                // hide cursor
	defer fmt.Fprint(u.out, "\r\x1b[2K\x1b[?25h") // clear the line, show cursor

	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()
	for i := 0; ; i++ {
		// Redraw the whole line so keys echoed by the terminal are wiped.
		fmt.Fprintf(u.out, "\r\x1b[2K%s %s", frameStyle.Render(spinnerFrames[i%len(spinnerFrames)]), title)
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
