package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"
)

// server side handling by the supervisor.
// NOTE: monitored process is not continued on errors!
func serverSide(supervisor, monitored int, conn net.Conn) error {
	log.Printf("server side supervisor (pid %d) hijacked %s -> %s from pid %d", supervisor,
		conn.RemoteAddr().String(), conn.LocalAddr().String(), monitored)
	msg := fmt.Sprintf("PONG from %d on %s -> %s\n", supervisor,
		conn.RemoteAddr().String(), conn.LocalAddr().String())

	defer conn.Close()

	rdr := bufio.NewReader(conn)
	data, err := rdr.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			log.Printf("supervisor (pid %d) failed to read data: %s\n", supervisor, err)
		}
	}
	log.Printf("supervisor (pid %d): %s\n", supervisor, string(data))
	_, err = conn.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("supervisor (pid %d) failed to send: %w", supervisor, err)
	}
	return syscall.Kill(monitored, syscall.SIGCONT)
}

// client side handling by the supervisor.
// NOTE: monitored process is not continued on errors!
func clientSide(supervisor, monitored int, conn net.Conn) error {
	log.Printf("client side supervisor (pid %d) hijacked %s -> %s from pid %d", supervisor,
		conn.LocalAddr().String(), conn.RemoteAddr().String(), monitored)
	msg := fmt.Sprintf("PING from pid %d on %s -> %s\n", supervisor,
		conn.RemoteAddr().String(), conn.LocalAddr().String())

	defer conn.Close()

	_, err := conn.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("supervisor (pid %d) failed to send: %w", supervisor, err)
	}

	rdr := bufio.NewReader(conn)
	data, err := rdr.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			log.Printf("supervisor (pid %d) failed to read data: %s\n", supervisor, err)
		}
	}
	log.Printf("supervisor (pid %d): %s\n", supervisor, string(data))
	return syscall.Kill(monitored, syscall.SIGCONT)
}
