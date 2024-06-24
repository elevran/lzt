package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"

	"github.com/elevran/lzt/pkg/rawfd"
)

var (
	server = flag.String("server", ":3333", "IP:Port to connect to.")
	pause  = flag.Bool("pause", false, "automatically pause after connecting to server.")
	count  = flag.Int("count", 1, "Number of times to send message to server.")
)

func main() {
	pid := syscall.Getpid()

	flag.Parse()

	conn, err := net.Dial("tcp", *server)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	fd, err := rawfd.FromTCPConn(conn)
	if err != nil {
		log.Fatalln("failed to get file descriptor:", err)
	}

	defer log.Println(pid, "closed connection", fd)

	log.Printf("client (PID %d) new connection %d (%s -> %s)\n", pid, fd,
		conn.LocalAddr().String(), conn.RemoteAddr().String())

	if *pause {
		log.Println("process", pid, "pausing - sending SIGSTOP to self")
		if err = syscall.Kill(pid, syscall.SIGSTOP); err != nil {
			log.Fatalln("process", pid, "failed to stop:", err)
		}
	}
	log.Println("process", pid, "continuing")

	reader := bufio.NewReader(conn)
	line := fmt.Sprintln("PING from", pid, conn.LocalAddr().String())

	for _ = range *count {
		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Println("error writing socket:", err)
			return
		}

		bytes, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			if err != io.EOF {
				fmt.Println("failed to read echoed data:", err)
			}
			return
		}
		if line != string(bytes) {
			log.Println("echo reply differs:", bytes)
		}
	}
	_, _ = conn.Write([]byte("STOP\n"))
}
