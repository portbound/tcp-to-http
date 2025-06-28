package server

import (
	"io"

	"github.com/portbound/tcp-to-http/internal/request"
)

type Handler func(w io.Writer, req *request.Request) *HandlerError
type HandlerError struct {
	StatusCode int
	Message    string
}
