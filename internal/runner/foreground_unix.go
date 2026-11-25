//go:build unix

package runner

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// reclaimTerminal makes our process group the terminal's foreground group again.
// An interactive shell moves the terminal to its own group and may leave it there.
func reclaimTerminal(f *os.File) {
	fd := int(f.Fd())
	pgrp, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err != nil || pgrp == unix.Getpgrp() {
		return
	}
	// From a background group, changing the foreground group raises SIGTTOU, which would stop us.
	signal.Ignore(syscall.SIGTTOU)
	defer signal.Reset(syscall.SIGTTOU)
	_ = unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, unix.Getpgrp())
}
