package wormhole

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// DialRelay connects to relay, sends RoomID, and returns the connection (piped to peer after match).
func DialRelay(relayAddr, code string) (net.Conn, error) {
	addr, err := ParseRelayAddr(relayAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	id := RoomID(code)
	if _, err := conn.Write(id[:]); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// SendFile sends a file through the wormhole.
func SendFile(relayAddr, code, filePath string, onProgress func(int64, int64)) error {
	conn, err := DialRelay(relayAddr, code)
	if err != nil {
		return err
	}
	defer conn.Close()

	secure, err := UpgradeConn(conn, code, true)
	if err != nil {
		return err
	}
	defer secure.Close()

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
	return nil
}

// SendText sends text through the wormhole.
func SendText(relayAddr, code, text string) error {
	conn, err := DialRelay(relayAddr, code)
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
	_, err = secure.Write([]byte(text))
	return err
}

// Receive receives data from the wormhole (file or text).
func Receive(relayAddr, code, outDir string, onProgress func(int64, int64)) error {
	conn, err := DialRelay(relayAddr, code)
	if err != nil {
		return err
	}
	defer conn.Close()

	secure, err := UpgradeConn(conn, code, false)
	if err != nil {
		return err
	}
	defer secure.Close()

	h, err := ReadMetaHeader(secure)
	if err != nil {
		return err
	}

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
		return nil

	case TypeText:
		data := make([]byte, h.Size)
		if _, err := io.ReadFull(secure, data); err != nil {
			return err
		}
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			Width(60)
		fmt.Println(box.Render(string(data)))
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
