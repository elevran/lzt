package rawfd

import (
	"fmt"
	"net"
)

const (
	INVALID = -1 // invalid file descriptor number
)

// FromConnection returns the file descriptor number associated with the connection
func FromConnection(conn net.Conn) (int, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return INVALID, fmt.Errorf("Not a TCP connection: %T", conn)
	}

	raw, err := tcpConn.SyscallConn()
	if err != nil {
		return INVALID, fmt.Errorf("Error getting raw connection: %w", err)
	}

	var fd int
	err = raw.Control(func(descriptor uintptr) {
		fd = int(descriptor)
	})
	if err != nil {
		return INVALID, fmt.Errorf("Error getting file descriptor: %w", err)
	}

	return fd, nil
}
