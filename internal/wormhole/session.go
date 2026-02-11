// Package wormhole implements a secure P2P tunnel using PAKE + AES-256-CTR.
//
// Usage example:
//
//	// Server (receiver)
//	ln, _ := net.Listen("tcp", ":9999")
//	conn, _ := ln.Accept()
//	secure, _ := wormhole.UpgradeDefault(conn, "shared-password", false)
//	defer secure.Close()
//	// secure.Read/Write are now encrypted
//
//	// Client (sender)
//	conn, _ := net.Dial("tcp", "localhost:9999")
//	secure, _ := wormhole.UpgradeDefault(conn, "shared-password", true)
//	defer secure.Close()
//	secure.Write([]byte("hello"))
package wormhole

import (
	"crypto/cipher"
	"io"
	"net"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
)

// SecureConn wraps a net.Conn with transparent encryption/decryption.
// It implements net.Conn.
type SecureConn struct {
	conn   net.Conn
	reader io.Reader
	writer io.Writer
}

// Upgrade runs the handshake, verification, and returns a secured connection.
// handshaker and streamCipher can be nil to use defaults.
func Upgrade(conn net.Conn, password string, isSender bool, handshaker Handshaker, streamCipher StreamCipher) (*SecureConn, error) {
	logger.Info("wormhole.Upgrade start", logger.Context("params", map[string]any{
		"local": conn.LocalAddr().String(), "remote": conn.RemoteAddr().String(),
		"is_sender": isSender, "password_len": len(password),
	})...)
	if handshaker == nil {
		handshaker = DefaultHandshaker
	}
	if streamCipher == nil {
		streamCipher = DefaultStreamCipher
	}

	transport := NewFrameTransport(conn)
	key, err := handshaker.Run(transport, password, isSender)
	if err != nil {
		return nil, err
	}
	logger.Debug("PAKE handshake completed")

	encStream, decStream, err := streamCipher.NewDuplex(key, isSender)
	if err != nil {
		return nil, err
	}

	// Phase 2: Ping-Pong verification
	if isSender {
		encrypted := make([]byte, len(MagicVerify))
		encStream.XORKeyStream(encrypted, []byte(MagicVerify))
		if err := transport.SendFrame(encrypted); err != nil {
			return nil, err
		}
	} else {
		frame, err := transport.ReadFrame()
		if err != nil {
			return nil, err
		}
		decrypted := make([]byte, len(frame))
		decStream.XORKeyStream(decrypted, frame)
		if string(decrypted) != MagicVerify {
			return nil, ErrVerifyFailed
		}
	}

	logger.Debug("verification passed, secure tunnel ready")
	// Phase 3: Transparent tunnel (no more framing; raw encrypted stream)
	// encStream: our writes; decStream: our reads
	rd := &cipher.StreamReader{S: decStream, R: conn}
	wr := &cipher.StreamWriter{S: encStream, W: conn}

	logger.Info("wormhole.Upgrade done", logger.Context("result", map[string]any{
		"local": conn.LocalAddr().String(), "remote": conn.RemoteAddr().String(),
	})...)
	return &SecureConn{conn: conn, reader: rd, writer: wr}, nil
}

// UpgradeDefault is a convenience that uses default handshaker and cipher.
func UpgradeDefault(conn net.Conn, password string, isSender bool) (*SecureConn, error) {
	return Upgrade(conn, password, isSender, nil, nil)
}

// Read decrypts data on the fly.
func (s *SecureConn) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

// Write encrypts data on the fly.
func (s *SecureConn) Write(p []byte) (n int, err error) {
	return s.writer.Write(p)
}

// Close closes the underlying connection.
func (s *SecureConn) Close() error {
	return s.conn.Close()
}

// LocalAddr returns the local network address.
func (s *SecureConn) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (s *SecureConn) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// SetDeadline sets the connection deadline.
func (s *SecureConn) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline.
func (s *SecureConn) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline.
func (s *SecureConn) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}
