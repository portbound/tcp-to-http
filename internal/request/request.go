package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

const bufferSize = 8
const (
	stateinit = iota
	stateDone
)

type Request struct {
	RequestLine RequestLine
	State       int
}
type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r *Request) parse(data []byte) (int, error) {
	if r.State == stateinit {
		return parseRequestLine(r, data)
	}

	if r.State == stateDone {
		return 0, fmt.Errorf("error: trying to read data in a done state")
	}

	return 0, fmt.Errorf("error: unknown state")
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize, bufferSize)
	cr := []byte{'\r', '\n'}
	readToIndex := 0
	req := Request{
		State: stateinit,
	}

	for req.State != stateDone {
		if readToIndex == cap(buf) {
			tmp := make([]byte, cap(buf)*2)
			copy(tmp, buf[:readToIndex])
			buf = tmp
		}

		bytesRead, err := reader.Read(buf[readToIndex:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				req.State = 1
				break
			}
			return nil, err
		}
		readToIndex += bytesRead

		if !bytes.Contains(buf, cr) {
			continue
		}

		eol := bytes.Index(buf, cr)

		_, err = req.parse(buf[:eol])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[(eol+len(cr)):readToIndex])
		readToIndex -= eol + len(cr)
	}
	return &req, nil
}

func parseRequestLine(r *Request, data []byte) (int, error) {
	fields := strings.Split(string(data), " ")

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

	return len(data), nil
}
