package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

func (h Headers) Get(key string) string {
	val, ok := h[strings.ToLower(key)]
	if !ok {
		return ""
	}
	return val
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	crlf := []byte{'\r', '\n'}

	if bytes.HasPrefix(data, crlf) {
		return len(crlf), true, nil
	}

	lineEnd := bytes.Index(data, crlf)
	line := data[:lineEnd]
	line = bytes.TrimSpace(line)
	colon := bytes.Index(line, []byte{':'})

	if line[colon] <= 0 {
		return 0, false, fmt.Errorf("invalid header: missing field name")
	}
	if line[colon-1] == byte(' ') {
		return 0, false, fmt.Errorf("invalid header: space before colon")
	}

	fieldName := string(line[:colon])
	for _, ch := range fieldName {
		if !isValidTokenChar(ch) {
			return 0, false, fmt.Errorf("invalid header: non ascii characters in field-name")
		}
	}

	fieldName = strings.ToLower(fieldName)
	fieldValue := string(bytes.TrimSpace(line[colon+1:]))

	value, ok := h[fieldName]
	if ok {
		h[fieldName] = fmt.Sprintf("%s, %s", value, fieldValue)
	} else {
		h[fieldName] = fieldValue
	}

	return len(line) + len(crlf), false, nil
}

func isValidTokenChar(ch rune) bool {
	switch {
	case 'A' <= ch && ch <= 'Z':
		return true
	case 'a' <= ch && ch <= 'z':
		return true
	case '0' <= ch && ch <= '9':
		return true
	case strings.ContainsRune("!#$%&'*+-.^_`|~", ch):
		return true
	default:
		return false
	}
}
