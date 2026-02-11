package wormhole

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

const roomIDLen = 4

// RoomID converts a code string to 4 bytes for the relay.
func RoomID(code string) [roomIDLen]byte {
	var id [roomIDLen]byte
	copy(id[:], code)
	return id
}

// ParseRelayAddr parses "tcp://host:port" to host:port.
func ParseRelayAddr(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "tcp://") {
		return addr[6:], nil
	}
	return addr, nil
}

// Role bytes for relay protocol (must match PAKE: 0=sender, 1=receiver).
const (
	RoleSender   = 0
	RoleReceiver = 1
)

// DialRelay connects to relay, sends RoomID+role, and returns the connection (piped to opposite role after match).
// Relay pairs sender only with receiver to avoid "can't have its own role" in PAKE.
func DialRelay(relayAddr, code string, isSender bool) (net.Conn, error) {
	role := RoleReceiver
	if isSender {
		role = RoleSender
	}
	logger.Info("wormhole.DialRelay start", logger.Context("params", map[string]any{
		"relay_addr": relayAddr, "code": code, "room_id_len": roomIDLen, "role": role,
	})...)
	addr, err := ParseRelayAddr(relayAddr)
	if err != nil {
		logger.Warn("wormhole.DialRelay parse failed", zap.Error(err), zap.String("relay_addr", relayAddr))
		return nil, err
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		logger.Warn("wormhole.DialRelay dial failed", zap.Error(err), zap.String("addr", addr), zap.String("code", code))
		return nil, err
	}
	id := RoomID(code)
	if _, err := conn.Write(id[:]); err != nil {
		conn.Close()
		logger.Warn("wormhole.DialRelay write room_id failed", zap.Error(err))
		return nil, err
	}
	if _, err := conn.Write([]byte{byte(role)}); err != nil {
		conn.Close()
		logger.Warn("wormhole.DialRelay write role failed", zap.Error(err))
		return nil, err
	}
	logger.Info("wormhole.DialRelay done", logger.Context("result", map[string]any{
		"addr": addr, "local": conn.LocalAddr().String(), "remote": conn.RemoteAddr().String(),
	})...)
	return conn, nil
}

// SendFile sends a file through the wormhole.
func SendFile(relayAddr, code, filePath string, onProgress func(int64, int64)) error {
	logger.Info("wormhole.SendFile start", logger.Context("params", map[string]any{
		"relay_addr": relayAddr, "code": code, "file_path": filePath, "has_progress_cb": onProgress != nil,
	})...)
	conn, err := DialRelay(relayAddr, code, true)
	if err != nil {
		return err
	}
	defer conn.Close()

	secure, err := UpgradeConn(conn, code, true)
	if err != nil {
		logger.Warn("wormhole.SendFile upgrade failed", zap.Error(err), zap.String("relay_addr", relayAddr), zap.String("code", code))
		return err
	}
	defer secure.Close()
	logger.Debug("wormhole.SendFile secure connection established")

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	name := filepath.Base(filePath)
	mode := uint32(0644)
	if info.Mode().IsRegular() {
		mode = uint32(info.Mode().Perm())
	}

	h := &MetaHeader{
		Type: TypeFile,
		Name: name,
		Size: info.Size(),
		Mode: mode,
	}
	if err := WriteMetaHeader(secure, h); err != nil {
		return err
	}
	logger.Info("wormhole.SendFile meta header sent", logger.Context("meta", map[string]any{
		"type": int(h.Type), "name": name, "size": info.Size(), "mode": mode,
	})...)

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var written int64
	buf := GetBuffer()
	defer PutBuffer(buf)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if _, wErr := secure.Write(buf[:n]); wErr != nil {
				return wErr
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written, info.Size())
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	logger.Info("wormhole.SendFile done", logger.Context("result", map[string]any{
		"file_path": filePath, "bytes_written": written, "total_size": info.Size(),
	})...)
	return nil
}

// SendText sends text through the wormhole.
func SendText(relayAddr, code, text string) error {
	logger.Info("wormhole.SendText start", logger.Context("params", map[string]any{
		"relay_addr": relayAddr, "code": code, "text_len": len(text),
	})...)
	conn, err := DialRelay(relayAddr, code, true)
	if err != nil {
		return err
	}
	defer conn.Close()

	secure, err := UpgradeConn(conn, code, true)
	if err != nil {
		return err
	}
	defer secure.Close()

	h := &MetaHeader{
		Type: TypeText,
		Size: int64(len(text)),
	}
	if err := WriteMetaHeader(secure, h); err != nil {
		return err
	}
	logger.Debug("wormhole.SendText meta header sent, writing body")
	_, err = secure.Write([]byte(text))
	if err != nil {
		logger.Warn("wormhole.SendText write failed", zap.Error(err))
		return err
	}
	logger.Info("wormhole.SendText done", logger.Context("result", map[string]any{"code": code, "bytes": len(text)})...)
	return nil
}

// ReceiveResult holds what was received (one of file or text per connection).
type ReceiveResult struct {
	FilePath string // set when TypeFile (saved path)
	Text     string // set when TypeText
}

// Receive receives data from the wormhole (file or text).
// If textResult is non-nil and payload is text, the received text is stored there.
// If result is non-nil, FilePath or Text is set so caller can show success info.
func Receive(relayAddr, code, outDir string, onProgress func(int64, int64), textResult *string, result *ReceiveResult) error {
	logger.Info("wormhole.Receive start", logger.Context("params", map[string]any{
		"relay_addr": relayAddr, "code": code, "out_dir": outDir, "has_progress_cb": onProgress != nil,
	})...)
	conn, err := DialRelay(relayAddr, code, false)
	if err != nil {
		return err
	}
	defer conn.Close()

	secure, err := UpgradeConn(conn, code, false)
	if err != nil {
		return err
	}
	defer secure.Close()
	logger.Debug("wormhole.Receive secure connection established")

	h, err := ReadMetaHeader(secure)
	if err != nil {
		logger.Warn("wormhole.Receive read meta failed", zap.Error(err))
		return err
	}
	logger.Info("wormhole.Receive meta header", logger.Context("meta", map[string]any{
		"type": int(h.Type), "name": h.Name, "size": h.Size, "mode": h.Mode,
	})...)

	switch h.Type {
	case TypeFile:
		outPath := filepath.Join(outDir, h.Name)
		if h.Name == "" {
			outPath = filepath.Join(outDir, "received")
		}
		f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode))
		if err != nil {
			return err
		}
		defer f.Close()

		var read int64
		buf := GetBuffer()
		defer PutBuffer(buf)
		for read < h.Size {
			toRead := int64(len(buf))
			if remain := h.Size - read; remain < toRead {
				toRead = remain
			}
			n, err := io.ReadFull(secure, buf[:toRead])
			if n > 0 {
				if _, wErr := f.Write(buf[:n]); wErr != nil {
					return wErr
				}
				read += int64(n)
				if onProgress != nil {
					onProgress(read, h.Size)
				}
			}
			if err != nil {
				if err == io.EOF && read == h.Size {
					break
				}
				return err
			}
		}
		logger.Info("wormhole.Receive file done", logger.Context("result", map[string]any{
			"out_path": outPath, "bytes_read": read, "total_size": h.Size,
		})...)
		if result != nil {
			result.FilePath = outPath
		}
		return nil

	case TypeText:
		logger.Debug("wormhole.Receive reading text body", logger.Context("params", map[string]any{"size": h.Size})...)
		data := make([]byte, h.Size)
		if _, err := io.ReadFull(secure, data); err != nil {
			return err
		}
		logger.Info("wormhole.Receive text done", logger.Context("result", map[string]any{"bytes": len(data)})...)
		s := string(data)
		if textResult != nil {
			*textResult = s
		}
		if result != nil {
			result.Text = s
		}
		if textResult == nil && result == nil {
			box := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#874BFD")).
				Padding(1, 2).
				Width(60)
			fmt.Println(box.Render(s))
		}
		return nil

	default:
		return fmt.Errorf("unknown payload type: %d", h.Type)
	}
}

// GenerateCode creates a random 4-character alphanumeric code.
func GenerateCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[i%len(chars)]
	}
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
