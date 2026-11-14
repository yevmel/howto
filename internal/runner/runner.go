// Package runner executes the confirmed command in the user's shell.
package runner

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// ShellPath returns shell if it can be found, otherwise /bin/sh.
func ShellPath(shell string) string {
	if shell != "" {
		if p, err := exec.LookPath(shell); err == nil {
			return p
		}
	}
	return "/bin/sh"
}

// Run executes command with shell -c (via sudo if asked), attached to the terminal, and returns its exit status.
// A command killed by a signal yields 128 + the signal number (130 for Ctrl-C).
func Run(shell, command string, sudo bool) (int, error) {
	// Catch (rather than ignore) SIGINT: ignored signals stay ignored in the child,
	// caught ones are reset to the default, so Ctrl-C reaches the command only.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	defer signal.Stop(sigs)

	args := []string{shell, "-c", command}
	if sudo {
		// Run the whole command line as root, not just its first word.
		args = append([]string{"sudo"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 0, err
	}
	if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal()), nil
	}
	return exitErr.ExitCode(), nil
}
