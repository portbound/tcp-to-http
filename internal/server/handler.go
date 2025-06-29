package server

import (
	"fmt"
	"io"

	"github.com/portbound/tcp-to-http/internal/request"
	"github.com/portbound/tcp-to-http/internal/response"
)

type Handler func(w io.Writer, req *request.Request) *HandlerError
type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

func (e *HandlerError) Write(w io.Writer) {
	fmt.Fprintf(w, "error in handler: %v", e)
}
