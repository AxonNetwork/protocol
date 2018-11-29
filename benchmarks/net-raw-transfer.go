package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

const PAYLOAD_SIZE = 20 * 1024 * 1024

func main() {
	var server bool
	if len(os.Args) == 1 {
		server = true
	}

	if server {
		listener, err := net.Listen("tcp4", ":9991")
		if err != nil {
			panic(err)
		}

		fmt.Println("[server] listening on", listener.Addr())

		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("[server] connection error:", err)
				continue
			}
			go handleConnection(conn)
		}

	} else {
		conn, err := net.Dial("tcp4", os.Args[1])
		if err != nil {
			panic(err)
		}
		defer conn.Close()

		f, err := os.Open("/dev/urandom")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		r := io.LimitReader(f, PAYLOAD_SIZE)

		n, err := io.Copy(conn, r)
		if err != nil && err != io.EOF {
			panic(err)
		} else if err == io.EOF {
		}
		fmt.Printf("[client] sent %v bytes\n", n)
		fmt.Println("[client] done")
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("[server] client connected: %v\n", conn.RemoteAddr())

	data := make([]byte, PAYLOAD_SIZE)
	var total int
	for {
		n, err := io.ReadFull(conn, data)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			continue
		} else if err != nil {
			panic(err)
		}

		total += n
	}

	fmt.Printf("[server] read %v bytes\n", total)
}
