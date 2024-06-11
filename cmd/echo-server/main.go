package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/elevran/lzt/pkg/rawfd"
)

var (
	listen = flag.String("listen", ":3333", "IP:Port to accept connections on.")
	sleep  = flag.Int("sleep", 0, "Duration to sleep after a connection is accepted.")
)

func main() {
	flag.Parse()

	l, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Panicln(err)
	}
	log.Printf("server (PID %d) listening on %s", os.Getpid(), *listen)
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	fd, err := rawfd.FromConnection(conn)
	if err != nil {
		log.Println("failed to get file descriptor:", err)
	}

	log.Printf("%d accepted connection %d (%s -> %s)\n", os.Getpid(),
		fd, conn.RemoteAddr().String(), conn.LocalAddr().String())

	if *sleep != 0 {
		log.Println(os.Getpid(), "sleeping for", *sleep, "seconds")
		time.Sleep(time.Duration(*sleep) * time.Second)
		log.Println(os.Getpid(), "done sleeping")
	}

	defer func() {
		_ = conn.Close()
		log.Println(os.Getpid(), "closed connection", fd)
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
