package headers

import "testing"

func NewHeaders() Headers {
	return make(Headers)
}

func TestHeadersParse(t *testing.T) {
	tests := []struct {
		name        string
		initial     Headers
		data        []byte
		expectErr   bool
		expectDone  bool
		expectBytes int
		expectVals  map[string]string
	}{
		{
			name:        "Valid single header",
			initial:     NewHeaders(),
			data:        []byte("Host: localhost:42069\r\n\r\n"),
			expectErr:   false,
			expectDone:  false,
			expectBytes: 23,
			expectVals: map[string]string{
				"host": "localhost:42069",
			},
		},
		{
			name:        "Valid header with extra whitespace",
			initial:     NewHeaders(),
			data:        []byte("       Host: localhost:42069       \r\n\r\n"),
			expectErr:   false,
			expectDone:  false,
			expectBytes: 23,
			expectVals: map[string]string{
				"host": "localhost:42069",
			},
		},
		// This test won't work since it's calling the Parse method directly. The loop for parsing exists outside this package inside of Request.
		// {
		// 	name: "Valid 2 headers with existing",
		// 	initial: Headers{
		// 		"host": "localhost:42069",
		// 	},
		// 	data:       []byte("Content-Type: application/json\r\nCache-Control: max-age=604800\r\n\r\n"),
		// 	expectErr:  false,
		// 	expectDone: false,
		// 	expectVals: map[string]string{
		// 		"host":          "localhost:42069",
		// 		"content-type":  "application/json",
		// 		"cache-control": "max-age=604800",
		// 	},
		// },
		{
			name: "Valid append header",
			initial: Headers{
				"set-person": "lane-loves-go, prime-loves-zig, tj-loves-ocaml",
			},
			data:       []byte("Set-Person: jake-loves-vim\r\n\r\n"),
			expectErr:  false,
			expectDone: false,
			expectVals: map[string]string{
				"set-person": "lane-loves-go, prime-loves-zig, tj-loves-ocaml, jake-loves-vim",
			},
		},
		{
			name:        "Valid done (blank line)",
			initial:     NewHeaders(),
			data:        []byte("\r\n"),
			expectErr:   false,
			expectDone:  true,
			expectBytes: 2,
		},
		{
			name:        "Invalid spacing in header",
			initial:     NewHeaders(),
			data:        []byte("       Host : localhost:42069       \r\n\r\n"),
			expectErr:   true,
			expectDone:  false,
			expectBytes: 0,
		},
		{
			name:        "Invalid character in field name",
			initial:     NewHeaders(),
			data:        []byte("H©st: localhost:42069\r\n\r\n"),
			expectErr:   true,
			expectDone:  false,
			expectBytes: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			headers := tc.initial
			n, done, err := headers.Parse(tc.data)

			if tc.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if done != tc.expectDone {
				t.Errorf("done: got %v, want %v", done, tc.expectDone)
			}
			if tc.expectBytes != 0 && n != tc.expectBytes {
				t.Errorf("bytes consumed: got %d, want %d", n, tc.expectBytes)
			}
			for k, expected := range tc.expectVals {
				got := headers[k]
				if got != expected {
					t.Errorf("header %q: got %q, want %q", k, got, expected)
				}
			}
		})
	}
}
