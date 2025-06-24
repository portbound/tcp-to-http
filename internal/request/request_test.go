package request

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

// Read reads up to len(p) or numBytesPerRead bytes from the string per call
// its useful for simulating reading a variable number of bytes per chunk from a network connection
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := cr.pos + cr.numBytesPerRead
	if endIndex > len(cr.data) {
		endIndex = len(cr.data)
	}
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n
	if n > cr.numBytesPerRead {
		n = cr.numBytesPerRead
		cr.pos -= n - cr.numBytesPerRead
	}
	return n, nil
}

func TestRequestLineParse(t *testing.T) {
	tests := []struct {
		name           string
		input          io.Reader
		expectError    bool
		expectedMethod string
		expectedTarget string
		expectedVer    string
	}{
		{
			name:           "Valid GET root",
			input:          &chunkReader{data: "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n", numBytesPerRead: 3},
			expectError:    false,
			expectedMethod: "GET",
			expectedTarget: "/",
			expectedVer:    "HTTP/1.1",
		},
		{
			name:           "Valid GET with path",
			input:          &chunkReader{data: "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n", numBytesPerRead: 1},
			expectError:    false,
			expectedMethod: "GET",
			expectedTarget: "/coffee",
			expectedVer:    "HTTP/1.1",
		},
		{
			name:        "Invalid request line",
			input:       strings.NewReader("/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := RequestFromReader(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if r == nil {
				t.Fatalf("expected non-nil request")
			}

			if r.RequestLine.Method != tc.expectedMethod {
				t.Errorf("got method %q, want %q", r.RequestLine.Method, tc.expectedMethod)
			}
			if r.RequestLine.RequestTarget != tc.expectedTarget {
				t.Errorf("got target %q, want %q", r.RequestLine.RequestTarget, tc.expectedTarget)
			}
			if r.RequestLine.HttpVersion != tc.expectedVer {
				t.Errorf("got version %q, want %q", r.RequestLine.HttpVersion, tc.expectedVer)
			}
		})
	}
}

func TestHeadersParse(t *testing.T) {
	tests := []struct {
		name        string
		input       io.Reader
		expectError bool
		expected    map[string]string
	}{
		{
			name: "Standard headers",
			input: &chunkReader{
				data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
				numBytesPerRead: 3,
			},
			expectError: false,
			expected: map[string]string{
				"host":       "localhost:42069",
				"user-agent": "curl/7.81.0",
				"accept":     "*/*",
			},
		},
		{
			name: "Malformed header",
			input: &chunkReader{
				data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
				numBytesPerRead: 3,
			},
			expectError: true,
		},
		{
			name: "Empty input",
			input: &chunkReader{
				data:            "",
				numBytesPerRead: 3,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := RequestFromReader(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if r == nil {
				t.Fatalf("expected non-nil request")
			}

			for key, want := range tc.expected {
				got := r.Headers[key]
				if !reflect.DeepEqual(got, want) {
					t.Errorf("header %q: got %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestBodyParse(t *testing.T) {
	tests := []struct {
		name                  string
		input                 io.Reader
		expectError           bool
		expectedContentLength int
		expectedBody          string
	}{
		{
			name: "Standard Body",
			input: &chunkReader{
				data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"Content-Length: 13\r\n" +
					"\r\n" +
					"hello world!\n",
				numBytesPerRead: 3,
			},
			expectedContentLength: 13,
			expectedBody:          "hello world!\n",
		},
		{
			name: "Body shorter than reported content length",
			input: &chunkReader{
				data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"Content-Length: 20\r\n" +
					"\r\n" +
					"partial content",
				numBytesPerRead: 3,
			},
			expectError:           true,
			expectedContentLength: 20,
		},
		{
			name: "Empty body content length 0",
			input: &chunkReader{
				data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"Content-Length: 0\r\n" +
					"\r\n" +
					"",
				numBytesPerRead: 8,
			},
			expectError:           false,
			expectedContentLength: 0,
		},
		{
			name: "Empty body no content length",
			input: &chunkReader{
				data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"\r\n" +
					"",
				numBytesPerRead: 8,
			},
			expectError:           false,
			expectedContentLength: 0,
		},
		{
			name: "No content length but body exists",
			input: &chunkReader{
				data: "POST /submit HTTP/1.1\r\n" +
					"Host: localhost:42069\r\n" +
					"\r\n" +
					"here is a body that shouldn't be read",
				numBytesPerRead: 8,
			},
			expectError:           false,
			expectedContentLength: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := RequestFromReader(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if r == nil {
				t.Fatalf("expected non-nil request")
			}

			if !reflect.DeepEqual(string(r.Body), tc.expectedBody) {
				t.Errorf("got body %q, want %q", string(r.Body), tc.expectedBody)
			}

			if len(r.Body) != tc.expectedContentLength {
				t.Errorf("got content length %q, want %q", r.Body, tc.expectedContentLength)
			}
		})
	}
}
