package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/portbound/tcp-to-http/internal/headers"
)

const bufferSize = 8
const (
	stateParsingRequestLine = iota
	stateParsingHeaders
	stateDone
)

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	State       int
}
type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r *Request) parse(buf []byte) (int, error) {
	if r.State == stateParsingRequestLine {
		return parseRequestLine(r, buf)
	}
	if r.State == stateParsingHeaders {
		bytesParsed, done, err := r.Headers.Parse(buf)
		if err != nil {
			return 0, err
		}

		if done {
			r.State = stateDone
		}
		return bytesParsed, nil
	}
	return 0, fmt.Errorf("error: unknown state")
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize, bufferSize)
	crlf := []byte{'\r', '\n'}
	index := 0
	req := Request{
		Headers: make(headers.Headers),
		State:   stateParsingRequestLine,
	}

	for req.State != stateDone {
		if index == cap(buf) {
			tmp := make([]byte, cap(buf)*2)
			copy(tmp, buf[:index])
			buf = tmp
		}

		bytesRead, err := reader.Read(buf[index:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				req.State = 1
				return nil, err
			}
			return nil, err
		}
		index += bytesRead

		if !bytes.Contains(buf, crlf) {
			continue
		}

		bytesParsed, err := req.parse(buf)
		if err != nil {
			return nil, err
		}

		tmp := make([]byte, len(buf))
		copy(tmp, buf[bytesParsed:index])
		buf = tmp

		index -= bytesParsed
	}
	return &req, nil
}

func parseRequestLine(r *Request, buf []byte) (int, error) {
	crlf := []byte{'\r', '\n'}

	lineEnd := bytes.Index(buf, crlf)
	line := string(buf[:lineEnd])
	fields := strings.Split(line, " ")

	if len(fields) != 3 {
		return 0, fmt.Errorf("invalid Request Line. Expected 3 parts, got %d, %v", len(fields), fields)
	}

	if fields[2] != "HTTP/1.1" {
		return 0, fmt.Errorf("only HTTP/1.1 is supported. Got=%s", fields[2])
	}

	for _, char := range fields[0] {
		if !unicode.IsUpper(char) {
			return 0, fmt.Errorf("invalid HTTP method. Got=%s", fields[0])
		}
	}

	r.RequestLine = RequestLine{
		HttpVersion:   fields[2],
		RequestTarget: fields[1],
		Method:        fields[0],
	}

	r.State = stateParsingHeaders
	return len(line) + len(crlf), nil
}
