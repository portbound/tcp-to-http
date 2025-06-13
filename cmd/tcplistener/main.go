package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

const port = ":42069"

func main() {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("error: failed to listen to TCP traffic: %s", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("error: failed to accept conn => %s", err)
		}

		fmt.Printf("Accepted connection from %s\n", conn.RemoteAddr())

		for line := range getLinesChannel(conn) {
			fmt.Println(line)
		}
		fmt.Printf("Connection to %s closed\n", conn.RemoteAddr())
	}
}

func getLinesChannel(conn io.ReadCloser) <-chan string {
	ch := make(chan string)

	go func() {
		defer conn.Close()
		defer close(ch)

		line := ""
		buffer := make([]byte, 8)

		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				fmt.Printf("error: %s\n", err.Error())
			}

			line += string(buffer[:n])

			parts := strings.Split(line, "\n")

			// each part except the last is guaranteed end on a '\n'
			// if the last part happens to end on '\n' it will be caught in next iteration nbd
			for _, part := range parts[:len(parts)-1] {
				ch <- part
			}
			line = parts[len(parts)-1]
		}

		// send any remaining text that didn't fill the buffer by 8 bytes and didn't end in a new line
		if line != "" {
			ch <- line
		}
	}()
	return ch
}
