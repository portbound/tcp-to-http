package server

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/portbound/tcp-to-http/internal/request"
	"github.com/portbound/tcp-to-http/internal/response"
)

type Server struct {
	handler  Handler
	listener net.Listener
	closed   atomic.Bool
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			log.Printf("error accepting connection: %v", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		handlerErr := HandlerError{
			StatusCode: response.StatusBadRequest,
			Message:    err.Error(),
		}
		handlerErr.Write(conn)
		return
	}

	buf := bytes.NewBuffer([]byte{})
	handlerErr := s.handler(buf, req)
	if handlerErr != nil {
		handlerErr.Write(conn)
		return
	}

	err = response.WriteStatusLine(buf, response.StatusOk)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	contentLen, err := strconv.Atoi(req.Headers.Get("Content-Length"))
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	defaultHeaders := response.GetDefaultHeaders(contentLen)

	err = response.WriteHeaders(buf, defaultHeaders)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	conn.Write([]byte("\r\n"))
}

func (s *Server) Close() error {
	if err := s.listener.Close(); err != nil {
		return err
	}
	return nil
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	s := Server{
		handler:  handler,
		listener: listener,
	}
	go s.listen()
	return &s, nil
}
