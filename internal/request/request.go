package request

import (
	"bytes"
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

type ReadResult struct {
	Data      []byte
	BytesRead int
	Err       error
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	dataChan := make(chan ReadResult)
	buf := make([]byte, 0, bufferSize)
	crlf := []byte{'\r', '\n'}
	req := Request{
		Headers: make(headers.Headers),
		State:   stateParsingRequestLine,
	}

	go func() {
		for {
			buf := make([]byte, bufferSize)
			n, err := reader.Read(buf)
			dataChan <- ReadResult{
				Data:      buf[:n],
				BytesRead: n,
				Err:       err,
			}
			if err != nil {
				close(dataChan)
				return
			}
		}
	}()

	for req.State != stateDone {
		stream, ok := <-dataChan
		if !ok {
			return nil, io.EOF
		}

		if len(buf)+stream.BytesRead > cap(buf) {
			tmp := make([]byte, 0, cap(buf)*2)
			tmp = append(tmp, buf...)
			buf = tmp
		}

		buf = append(buf, stream.Data[:stream.BytesRead]...)

		if req.State == stateParsingRequestLine || req.State == stateParsingHeaders {
			if !bytes.Contains(buf, crlf) {
				continue
			}
		}

		if req.State == stateParsingBody {
			if len(buf) != req.ContentLength {
				continue
			}
		}

		bytesParsed, err := req.parse(buf)
		if err != nil {
			return nil, err
		}

		if req.State == stateDone {
			break
		}

		tmp := make([]byte, 0, len(buf))
		tmp = append(tmp, buf[bytesParsed:]...)
		buf = tmp
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
