package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/portbound/tcp-to-http/internal/response"
)

type Server struct {
	Listener net.Listener
	closed   atomic.Bool
}

func (s *Server) listen() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if s.closed.Load() != true {
				return
			}
			log.Printf("error: %v", err)
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	err := response.WriteStatusLine(conn, response.StatusOk)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	defaultHeaders := response.GetDefaultHeaders(0)
	err = response.WriteHeaders(conn, defaultHeaders)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	conn.Write([]byte("\r\n"))
}

func (s *Server) Close() error {
	if err := s.Listener.Close(); err != nil {
		return err
	}
	return nil
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := Server{
		Listener: listener,
	}

	go server.listen()

	return &server, nil
}
