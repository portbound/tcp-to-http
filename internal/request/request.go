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
	Header      headers.Headers
	State       int
}
type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r *Request) parse(line []byte) (int, error) {
	if r.State == stateParsingRequestLine {
		return parseRequestLine(r, line)
	}

	if r.State == stateDone {
		return 0, fmt.Errorf("error: trying to read data in a done state")
	}

	return 0, fmt.Errorf("error: unknown state")
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize, bufferSize)
	crlf := []byte{'\r', '\n'}
	index := 0
	req := Request{
		State: stateParsingRequestLine,
	}

	for req.State == stateParsingRequestLine {
		if index == cap(buf) {
			tmp := make([]byte, cap(buf)*2)
			copy(tmp, buf[:index])
			buf = tmp
		}

		bytesRead, err := reader.Read(buf[index:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				req.State = 1
				break
			}
			return nil, err
		}
		index += bytesRead

		if !bytes.Contains(buf, crlf) {
			continue
		}

		lineEnd := bytes.Index(buf, crlf)
		line := buf[:lineEnd]

		_, err = req.parse(line)
		if err != nil {
			return nil, err
		}

		copy(buf, buf[(lineEnd+len(crlf)):index])
		index -= lineEnd + len(crlf)

		req.State = stateParsingHeaders
	}

	for req.State == stateParsingHeaders {
		fmt.Println("here")
		headers := make(headers.Headers)

		bytesRead, err := reader.Read(buf[index:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				req.State = 1
				break
			}
			return nil, err
		}
		index += bytesRead

		if !bytes.Contains(buf, crlf) {
			continue
		}

		lineEnd := bytes.Index(buf, crlf)
		line := buf[:lineEnd]

		_, done, err := headers.Parse(line)
		if err != nil {
			return nil, err
		}

		if done == true {
			req.Header = headers
			req.State = stateDone
		}
		copy(buf, buf[(lineEnd+len(crlf)):index])
		index -= lineEnd + len(crlf)

	}

	fmt.Printf("%v", &req)
	return &req, nil
}

func parseRequestLine(r *Request, line []byte) (int, error) {
	fields := strings.Split(string(line), " ")

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

	r.State = stateDone
	r.RequestLine = RequestLine{
		HttpVersion:   fields[2],
		RequestTarget: fields[1],
		Method:        fields[0],
	}

	return len(line), nil
}
