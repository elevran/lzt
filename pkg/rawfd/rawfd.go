package rawfd

import (
	"fmt"
	"net"
	"os"
)

const (
	INVALID = -1 // invalid file descriptor number
)

// FromConn returns the file descriptor number associated with the connection
func FromTCPConn(conn net.Conn) (int, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return INVALID, fmt.Errorf("not a TCP connection: %T", conn)
	}

	raw, err := tcpConn.SyscallConn()
	if err != nil {
		return INVALID, fmt.Errorf("failed getting raw connection: %w", err)
	}

	var fd int
	err = raw.Control(func(descriptor uintptr) {
		fd = int(descriptor)
	})
	if err != nil {
		return INVALID, fmt.Errorf("failed getting file descriptor: %w", err)
	}

	return fd, nil
}

// ToTCPConn returns a net.Conn corresponding to file descriptor
func ToTCPConn(fd int) (net.Conn, error) {
	file := os.NewFile(uintptr(fd), "duplicated-fd")
	if file == nil {
		return nil, fmt.Errorf("os.NewFile(%d) failed", fd)
	}

	fc, err := net.FileConn(file)
	if err != nil {
		return nil, fmt.Errorf("failed to convert fd %d to FileCon: %w", fd, err)
	}

	tcp, ok := fc.(*net.TCPConn)
	if !ok {
		fc.Close()
		return nil, fmt.Errorf("fd %d isn't a TCP connection", fd)
	}
	return tcp, nil
}
