package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
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

	ca   = flag.String("ca", "", "CA certificate file (PEM encoded)")
	cert = flag.String("cert", "", "workload certificate file (PEM encoded)")
	key  = flag.String("key", "", "workload private key file (PEM encoded)")
)

func main() {
	flag.Parse()
	supervisorPID := syscall.Getpid()

	if err := validateCapabilities(supervisorPID); err != nil {
		log.Fatalln("invalid capabilities:", err)
	}

	if err := validateOptions(); err != nil {
		log.Fatalln("invalid options:", err)
	}

	conn, err := duplicateProcessDescriptor(supervisorPID, *pid, *fd)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	if *ca != "" || *cert != "" || *key != "" { // TLS configuration defined
		tlsConfig, err := loadTLSConfig(*ca, *cert, *key)
		if err != nil {
			log.Fatalln("failed to load TLS configuration: %w", err)
		}

		if *isServer {
			tlsServerSide(supervisorPID, *pid, conn, tlsConfig)
		} else {
			tlsClientSide(supervisorPID, *pid, conn, tlsConfig)
		}
		return
	}

	// no TLS files provided, use regular TCP exchange
	if *isServer {
		serverSide(supervisorPID, *pid, conn)
	} else {
		clientSide(supervisorPID, *pid, conn)
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
	} else if *ca != "" || *cert != "" || *key != "" { // TLS configuration defined
		if !fileReadable(*ca) || !fileReadable(*cert) || !fileReadable(*key) {
			return errors.New("valid TLS configuration required")
		}
	}
	return nil
}

func fileReadable(path string) bool {
	f, err := os.Open(path)
	defer func() {
		_ = f.Close()
	}()
	return err == nil
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
