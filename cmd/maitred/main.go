package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/elevran/lzt/pkg/capability"
	"github.com/elevran/lzt/pkg/pidfd"
	"github.com/elevran/lzt/pkg/rawfd"
)

const (
	INVALID_PID = -1
	INVALID_FD  = rawfd.INVALID
)

var (
	pid      = flag.Int("pid", INVALID_PID, "The pid of the monitored process")
	fd       = flag.Int("fd", INVALID_FD, "The file descriptor to duplicate")
	isServer = flag.Bool("server", false, "Set if the monitored process runs as server")
)

func main() {
	flag.Parse()
	supervisor := syscall.Getpid()

	if err := validateCapabilities(supervisor); err != nil {
		log.Fatalln("invalid capabilities:", err)
	}

	if err := validateOptions(); err != nil {
		log.Fatalln("invalid options:", err)
	}

	conn, err := duplicateProcessDescriptor(supervisor, *pid, *fd)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	if *isServer {
		serverSide(supervisor, *pid, conn)
	} else {
		clientSide(supervisor, *pid, conn)
	}
}

// ensures the process posses required capabilities to affect the
// monitored process
func validateCapabilities(pid int) error {
	for _, cap := range []int{unix.CAP_SYS_PTRACE, unix.CAP_KILL} {
		enabled, err := capability.Has(cap)
		if err != nil {
			return err
		}
		if !enabled {
			return fmt.Errorf("supervisor process %d missing capability %d", pid, cap)
		}
	}
	return nil
}

// check if process has needed input (valid pid and fd numbers)
func validateOptions() error {
	if *pid <= 0 {
		return errors.New("valid pid required")
	} else if *fd == INVALID_FD || *fd < 0 {
		return errors.New("valid fd required")
	}
	return nil
}

// duplicate the provided fd from the monitored process
func duplicateProcessDescriptor(supervisor, monitored, fd int) (net.Conn, error) {
	log.Printf("supervisor (pid %d) duplicating fd %d from pid %d",
		supervisor, fd, monitored)

	pf, err := pidfd.Open(monitored)
	if err != nil {
		return nil, fmt.Errorf("pidfd.Open(%d) failed: %w", monitored, err)
	}
	defer pf.Close()

	newfd, err := pf.Get(fd)
	if err != nil {
		return nil, fmt.Errorf("pidfd(%d).Get(%d) failed: %w", monitored, fd, err)
	}
	return rawfd.ToTCPConn(newfd)
}

// server side handling by the supervisor.
// NOTE: monitored process is not continued on errors!
func serverSide(supervisor, monitored int, conn net.Conn) error {
	log.Printf("supervisor (pid %d) hijacked %s -> %s from pid %d", supervisor,
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
