package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

var megabytes = flag.Int("mb", 10, "megabytes to transfer")
var shouldWrite = flag.Bool("write", false, "write instead of read")

// const PAYLOAD_SIZE = 1024 * 1024 * 1024

func main() {
	flag.Parse()

	var server bool
	if len(flag.Args()) == 0 {
		server = true
	}

	var operation func(net.Conn) = read
	if *shouldWrite {
		operation = write
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
			fmt.Printf("client connected: %v\n", conn.RemoteAddr())

			stopwatch(func() { operation(conn) })
		}

	} else {
		conn, err := net.Dial("tcp4", flag.Args()[0])
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		fmt.Println("connected")

		stopwatch(func() { operation(conn) })
	}
}

func write(conn net.Conn) {
	defer conn.Close()

	f, err := os.Open("/dev/urandom")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := io.LimitReader(f, int64((*megabytes)*1024*1024))

	n, err := io.Copy(conn, r)
	if err != nil && err != io.EOF {
		panic(err)
	} else if err == io.EOF {
	}
	fmt.Printf("sent %v bytes\n", n)
	fmt.Println("done")
}

func read(conn net.Conn) {
	defer conn.Close()

	data := make([]byte, (*megabytes)*1024*1024)
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

func stopwatch(fn func()) {
	start := time.Now()
	fn()
	duration := time.Now().Sub(start)
	fmt.Println("took", duration.Seconds(), "seconds")
}
