package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/elevran/lzt/pkg/rawfd"
)

var (
	server = flag.String("server", ":3333", "IP:Port to connect to.")
	sleep  = flag.Int("sleep", 0, "Duration to sleep after a connection is accepted.")
	count  = flag.Int("count", 1, "Number of times to send message to server.")
)

func main() {
	flag.Parse()

	conn, err := net.Dial("tcp", *server)
	if err != nil {
		log.Panicln(err)
	}
	defer conn.Close()

	fd, err := rawfd.FromConnection(conn)
	if err != nil {
		log.Println("failed to get file descriptor:", err)
	}

	defer log.Println(os.Getpid(), "closed connection", fd)

	log.Printf("client (PID %d) new connection %d (%s -> %s)\n", os.Getpid(),
		fd, conn.LocalAddr().String(), conn.RemoteAddr().String())

	if *sleep != 0 {
		log.Println(os.Getpid(), "sleeping for", *sleep, "seconds")
		time.Sleep(time.Duration(*sleep) * time.Second)
		log.Println(os.Getpid(), "done sleeping")
	}

	reader := bufio.NewReader(conn)
	line := fmt.Sprintln("PING from", conn.LocalAddr().String())

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
