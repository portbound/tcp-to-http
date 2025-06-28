package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	httpServer "github.com/portbound/tcp-to-http/internal/server"
)

const port = 42069

func main() {
	s, err := httpServer.Serve(port)
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
