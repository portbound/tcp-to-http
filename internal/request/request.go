package request

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

// GET /coffee HTTP/1.1
// Host: localhost:42069
// User-Agent: curl/7.81.0
// Accept: */*

// {"flavor":"dark mode"}

type Request struct {
	RequestLine RequestLine
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := Request{}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read from request: %w", err)
	}

	rawRequest := strings.Split(string(data), "\r\n")

	if len(rawRequest) == 0 {
		return nil, fmt.Errorf("invalid request, missing Request Line")
	}

	reqLine, err := parseRequestLine(rawRequest[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse request line: %w", err)
	}

	req.RequestLine = *reqLine
	return &req, nil
}

func parseRequestLine(data string) (*RequestLine, error) {
	fields := strings.Split(data, " ")

	if len(fields) != 3 {
		return nil, fmt.Errorf("invalid Request Line. Expected 3 parts, got %d", len(fields))
	}

	reqLine := RequestLine{
		HttpVersion:   fields[2],
		RequestTarget: fields[1],
		Method:        fields[0],
	}

	if reqLine.HttpVersion != "HTTP/1.1" {
		return nil, fmt.Errorf("only HTTP/1.1 is supported. Got=%s", reqLine.HttpVersion)
	}

	for _, char := range reqLine.Method {
		if !unicode.IsUpper(char) {
			return nil, fmt.Errorf("invalid HTTP method. Got=%s", reqLine.Method)
		}
	}

	return &reqLine, nil
}
