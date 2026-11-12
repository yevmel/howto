package ui

import "golang.org/x/sys/unix"

// flushInput discards unread terminal input; TCIFLUSH equals FREAD for TIOCFLUSH.
func flushInput(fd int) error {
	return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, unix.TCIFLUSH)
}
