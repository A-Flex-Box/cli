package wormhole

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
)

const peekSize = 512

// TrafficInfo holds parsed protocol info from sniffed bytes.
type TrafficInfo struct {
	Protocol string // "HTTP", "SSH", "TCP"
	Method   string // HTTP: GET, POST, etc.
	Path     string // HTTP: /path
	Raw      string // Short display string
}

// analyzeTraffic peeks at the first bytes and returns protocol info.
func analyzeTraffic(peek []byte) TrafficInfo {
	if len(peek) == 0 {
		return TrafficInfo{Protocol: "TCP", Raw: "(no data)"}
	}

	// SSH: starts with "SSH-"
	if len(peek) >= 4 && string(peek[:4]) == "SSH-" {
		return TrafficInfo{
			Protocol: "SSH",
			Raw:      "[SSH] connection",
		}
	}

	// HTTP: method + space + path
	if idx := bytes.IndexByte(peek, '\n'); idx >= 0 {
		line := string(bytes.TrimSpace(peek[:idx]))
		parts := strings.SplitN(line, " ", 3)
		if len(parts) >= 2 {
			method := strings.ToUpper(parts[0])
			path := parts[1]
			// Common HTTP methods
			for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"} {
				if method == m {
					return TrafficInfo{
						Protocol: "HTTP",
						Method:   method,
						Path:     path,
						Raw:      "[HTTP] " + method + " " + path,
					}
				}
			}
		}
	}

	// Try http parse as fallback
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(peek)))
	if err == nil {
		return TrafficInfo{
			Protocol: "HTTP",
			Method:   req.Method,
			Path:     req.URL.Path,
			Raw:      "[HTTP] " + req.Method + " " + req.URL.Path,
		}
	}

	return TrafficInfo{Protocol: "TCP", Raw: "(binary stream)"}
}

// SniffingReader wraps a reader, peeks at the first peekSize bytes, analyzes
// protocol, and optionally invokes onEvent. Then serves all data transparently.
type SniffingReader struct {
	r       io.Reader
	prefix  []byte
	pos     int
	emitted bool
	onEvent func(TrafficInfo)
}

// NewSniffingReader creates a reader that sniffs the first chunk and calls onEvent.
// onEvent may be nil.
func NewSniffingReader(r io.Reader, onEvent func(TrafficInfo)) *SniffingReader {
	return &SniffingReader{r: r, onEvent: onEvent}
}

// Read implements io.Reader. On first read, peeks up to peekSize bytes, analyzes
// protocol, invokes onEvent, then serves data transparently.
func (s *SniffingReader) Read(p []byte) (n int, err error) {
	if s.pos < len(s.prefix) {
		n = copy(p, s.prefix[s.pos:])
		s.pos += n
		return n, nil
	}

	if !s.emitted {
		s.emitted = true
		peek := make([]byte, peekSize)
		nn, readErr := s.r.Read(peek)
		if nn > 0 {
			peek = peek[:nn]
			info := analyzeTraffic(peek)
			if s.onEvent != nil {
				s.onEvent(info)
			}
			s.prefix = peek
			s.pos = 0
			n = copy(p, s.prefix)
			s.pos = n
			return n, nil
		}
		if readErr != nil {
			return 0, readErr
		}
	}

	return s.r.Read(p)
}
