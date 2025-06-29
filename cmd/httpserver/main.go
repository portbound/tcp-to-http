package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/portbound/tcp-to-http/internal/request"
	httpServer "github.com/portbound/tcp-to-http/internal/server"
)

const port = 42069

func main() {
	s, err := httpServer.Serve(port, func(w io.Writer, req *request.Request) *httpServer.HandlerError {
		if req.RequestLine.RequestTarget == "/yourproblem" {
			return &httpServer.HandlerError{StatusCode: 400, Message: "Your problem is not my problem\n"}
		}

		if req.RequestLine.RequestTarget == "/myproblem" {
			return &httpServer.HandlerError{StatusCode: 500, Message: "Whoopsie, my bad\n"}
		}

		fmt.Fprintf(w, "All good, frfr\n")
		return nil
	})
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer s.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
