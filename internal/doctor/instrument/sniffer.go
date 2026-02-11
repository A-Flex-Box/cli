package instrument

import (
	"bytes"
	"strings"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

const (
	// MaxHeaderBytes is the maximum bytes used for protocol detection.
	MaxHeaderBytes = 512
)

// Protocol constants for DetectProtocol result.
const (
	ProtocolHTTP  = "HTTP"
	ProtocolSSH   = "SSH"
	ProtocolRedis = "Redis"
	ProtocolTLS   = "TLS/SSL"
	ProtocolTCP   = "TCP (Raw)"
)

// DetectProtocol analyzes the first bytes of a stream (up to MaxHeaderBytes) and returns
// the guessed protocol. Used for Deep Packet Inspection (DPI) logic.
func DetectProtocol(head []byte) string {
	if len(head) == 0 {
		logger.Debug("DetectProtocol: empty head, returning TCP (Raw)",
			zap.String("component", "instrument.Sniffer"))
		return ProtocolTCP
	}
	// TLS/SSL: magic byte 0x16 for Handshake
	if head[0] == 0x16 {
		logger.Debug("DetectProtocol: TLS/SSL handshake magic 0x16 detected",
			zap.String("component", "instrument.Sniffer"))
		return ProtocolTLS
	}
	// Normalize to ASCII for string checks
	headStr := strings.TrimSpace(string(head))
	upper := strings.ToUpper(headStr)
	// HTTP: GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS, CONNECT, or response HTTP/
	if strings.HasPrefix(upper, "GET ") ||
		strings.HasPrefix(upper, "POST ") ||
		strings.HasPrefix(upper, "HEAD ") ||
		strings.HasPrefix(upper, "PUT ") ||
		strings.HasPrefix(upper, "DELETE ") ||
		strings.HasPrefix(upper, "PATCH ") ||
		strings.HasPrefix(upper, "OPTIONS ") ||
		strings.HasPrefix(upper, "CONNECT ") ||
		strings.HasPrefix(upper, "HTTP/") {
		logger.Debug("DetectProtocol: HTTP signature detected",
			zap.String("component", "instrument.Sniffer"),
			zap.String("prefix", headStr[:min(20, len(headStr))]))
		return ProtocolHTTP
	}
	// SSH: starts with SSH-
	if strings.HasPrefix(headStr, "SSH-") {
		logger.Debug("DetectProtocol: SSH banner detected",
			zap.String("component", "instrument.Sniffer"))
		return ProtocolSSH
	}
	// Redis: Array (*), String (+), or PING
	if strings.HasPrefix(headStr, "*") || strings.HasPrefix(headStr, "+") ||
		strings.HasPrefix(upper, "PING") {
		logger.Debug("DetectProtocol: Redis signature detected",
			zap.String("component", "instrument.Sniffer"))
		return ProtocolRedis
	}
	logger.Debug("DetectProtocol: no signature match, returning TCP (Raw)",
		zap.String("component", "instrument.Sniffer"),
		zap.Int("head_len", len(head)))
	return ProtocolTCP
}

// SniffFromReader reads up to MaxHeaderBytes from r and runs DetectProtocol.
// Does not consume from r if you need to replay; caller may use io.MultiReader with a Buffer.
func SniffFromReader(r interface{ Read([]byte) (int, error) }) (protocol string, header []byte, err error) {
	buf := make([]byte, MaxHeaderBytes)
	n, err := r.Read(buf)
	if n > 0 {
		header = buf[:n]
		protocol = DetectProtocol(header)
		logger.Debug("SniffFromReader completed",
			zap.String("component", "instrument.Sniffer"),
			zap.String("protocol", protocol),
			zap.Int("bytes_read", n))
		return protocol, header, err
	}
	return ProtocolTCP, nil, err
}

// PeekProtocol is a helper: given raw bytes (e.g. from GrabBanner), returns the protocol.
func PeekProtocol(data []byte) string {
	return DetectProtocol(data)
}

// TruncateForDisplay truncates a byte slice for safe display in logs/UI.
func TruncateForDisplay(b []byte, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 64
	}
	s := string(bytes.TrimSpace(b))
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
}
