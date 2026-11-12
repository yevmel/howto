package ui

import "golang.org/x/sys/unix"

// flushInput discards unread terminal input.
func flushInput(fd int) error {
	return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH)
}
