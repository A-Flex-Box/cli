package wormhole

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

// MetaHeader is the first encrypted frame after handshake.
type MetaHeader struct {
	Type PayloadType `json:"t"`
	Name string     `json:"n,omitempty"` // Filename (for files)
	Size int64      `json:"s"`           // Total bytes
	Mode uint32     `json:"m,omitempty"` // File permission (e.g. 0644)
}

// FrameTransportImpl implements FrameTransport over io.ReadWriter.
type FrameTransportImpl struct {
	rw io.ReadWriter
}

// NewFrameTransport creates a FrameTransport backed by io.ReadWriter.
func NewFrameTransport(rw io.ReadWriter) FrameTransport {
	return &FrameTransportImpl{rw: rw}
}

// SendFrame writes uint32(len) + data.
func (f *FrameTransportImpl) SendFrame(data []byte) error {
	return sendFrame(f.rw, data)
}

// ReadFrame reads uint32(len) + data.
func (f *FrameTransportImpl) ReadFrame() ([]byte, error) {
	return readFrame(f.rw)
}

// sendFrame writes length-prefixed data (uint32 big-endian + payload).
func sendFrame(w io.Writer, data []byte) error {
	ln := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, ln); err != nil {
		return err
	}
	if ln > 0 {
		_, err := w.Write(data)
		return err
	}
	return nil
}

// readFrame reads length-prefixed data.
func readFrame(r io.Reader) ([]byte, error) {
	var ln uint32
	if err := binary.Read(r, binary.BigEndian, &ln); err != nil {
		return nil, err
	}
	if ln > 16*1024*1024 { // 16MB max
		return nil, ErrInvalidFrameSize
	}
	buf := make([]byte, ln)
	if ln == 0 {
		return buf, nil
	}
	_, err := io.ReadFull(r, buf)
	return buf, err
}

// WriteMetaHeader writes a length-prefixed JSON-encoded MetaHeader.
func WriteMetaHeader(w io.Writer, h *MetaHeader) error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	return sendFrame(w, data)
}

// ReadMetaHeader reads a length-prefixed JSON-encoded MetaHeader.
func ReadMetaHeader(r io.Reader) (*MetaHeader, error) {
	data, err := readFrame(r)
	if err != nil {
		return nil, err
	}
	var h MetaHeader
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}
