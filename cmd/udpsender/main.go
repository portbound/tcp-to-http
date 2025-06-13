package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

const server = "localhost:42069"

func main() {
	var (
		err error
	)
	addr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		log.Fatalf("error => failed to listen to UDP traffic: %s", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("error => failed to set up UDP conn: %s", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		str, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("error => failed to read line:  %s", err)
		}

		_, err = conn.Write([]byte(str))
		if err != nil {
			log.Fatalf("error => failed to write to UDP connection: %s", err)
		}
	}
}
