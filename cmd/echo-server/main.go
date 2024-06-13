package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"strings"
	"syscall"

	"github.com/elevran/lzt/pkg/rawfd"
)

var (
	listen = flag.String("listen", ":3333", "IP:Port to accept connections on.")
)

func main() {
	pid := syscall.Getpid()

	flag.Parse()

	l, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Panicln(err)
	}
	log.Printf("server (PID %d) listening on %s", pid, *listen)
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleRequest(pid, conn)
	}
}

func handleRequest(pid int, conn net.Conn) {
	fd, err := rawfd.FromTCPConn(conn)
	if err != nil {
		_ = conn.Close()
		log.Println("failed to get file descriptor:", err)
		return
	}

	log.Printf("process %d accepted connection %d (%s -> %s)\n", pid, fd,
		conn.RemoteAddr().String(), conn.LocalAddr().String())

	log.Println("process", pid, "sending SIGSTOP to self")
	if err = syscall.Kill(pid, syscall.SIGSTOP); err != nil {
		log.Fatalln("process", pid, "failed to stop:", err)
	}
	log.Println("process", pid, "continuing")

	defer func() {
		_ = conn.Close()
		log.Println(pid, "closed connection", fd)
	}()

	connReader := bufio.NewReader(conn)
	for {
		data, err := connReader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println("failed to read data:", err)
			}
			return
		}
		request := strings.TrimSpace(string(data))

		if request == "STOP" {
			break
		}
		conn.Write([]byte(data))
	}
}
