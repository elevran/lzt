package pidfd

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// PidFd, a file descriptor that refers to a process.
type PidFd int

// Open obtains a file descriptor that refers to a process.
func Open(pid int) (PidFd, error) {
	flags := uint(0)

	fd, _, errno := syscall.Syscall(unix.SYS_PIDFD_OPEN, uintptr(pid), uintptr(flags), 0)
	if errno != 0 {
		return 0, errno
	}
	return PidFd(fd), nil
}

// Get retrieves the specified file descriptor from another process.
func (fd PidFd) Get(targetfd int) (int, error) {
	flags := uint(0)

	newfd, _, err := syscall.Syscall(unix.SYS_PIDFD_GETFD, uintptr(fd), uintptr(targetfd), uintptr(flags))
	if err != 0 {
		return -1, err
	}
	return int(newfd), nil
}

// Close closes the process file handle
func (fd PidFd) Close() {
	syscall.Close(int(fd))
}
