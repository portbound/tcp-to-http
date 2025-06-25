package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/portbound/tcp-to-http/internal/headers"
)

const bufferSize = 8
const (
	stateParsingRequestLine = iota
	stateParsingHeaders
	stateParsingBody
	stateDone
)

type Request struct {
	RequestLine   RequestLine
	Headers       headers.Headers
	Body          []byte
	ContentLength int
	State         int
}
type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
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
		// FIXME: instead of trying to check conditions here, we should probably just fire off the read in a goroutine. If there's content to be read, we can read it, but the main thread will not be blocked, so now is a great time to slow down and explore concurrency in Go - we can probably leverage go routines and channels with the select statement to read if there's content available and continue the loop otherwise?

		if index == 0 || !bytes.Contains(buf, []byte{'\n'}) {
			if index == cap(buf) {
				tmp := make([]byte, cap(buf)*2)
				copy(tmp, buf[:index])
				buf = tmp
			}

			bytesRead, err := reader.Read(buf[index:])
			if err != nil {
				if errors.Is(err, io.EOF) {
					if req.State == stateDone {
						break
					}
					return nil, err
				}
				return nil, err
			}

			index += bytesRead
		}

		if !bytes.Contains(buf, crlf) {
			continue
		}

		bytesParsed, err := req.parse(buf)
		if err != nil {
			return nil, err
		}

		if req.State == stateDone {
			break
		}

		tmp := make([]byte, len(buf))
		copy(tmp, buf[bytesParsed:index])
		buf = tmp

		index -= bytesParsed
	}
	return &req, nil
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

		if !done {
			return bytesParsed, nil
		}

		if val := r.Headers.Get("Content-Length"); val != "0" {
			r.State = stateParsingBody
			r.ContentLength, err = strconv.Atoi(val)
			if err != nil {
				return bytesParsed, err
			}
			return bytesParsed, nil
		}

		r.State = stateDone
		return bytesParsed, nil
	}

	if r.State == stateParsingBody {
		value := r.Headers.Get("Content-Length")

		contentLen, err := strconv.Atoi(value)
		if err != nil {
			return 0, err
		}

		bytesParsed, err := parseBody(r, buf, contentLen)
		if err != nil {
			return 0, err
		}
		return bytesParsed, nil
	}
	return 0, fmt.Errorf("error: unknown state")
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

func parseBody(r *Request, buf []byte, contentLen int) (int, error) {
	for _, b := range buf {
		if b != byte(0) {
			r.Body = append(r.Body, b)
		}
	}

	if len(r.Body) != contentLen {
		return 0, fmt.Errorf("error: length of body %q does not match specified content length %q", len(r.Body), contentLen)
	}

	r.State = stateDone
	return contentLen, nil
}
