package wormhole

import "errors"

const (
	// MagicVerify is sent after handshake to verify the channel is secure.
	MagicVerify = "WORMHOLE_OK"
	// CurveSIEC is the curve name for PAKE (siec is fast and secure).
	CurveSIEC = "siec"
	// frameLenBytes is the length prefix size for frames.
	frameLenBytes = 4
)

// PayloadType indicates the type of data being transferred.
type PayloadType uint8

const (
	TypeFile PayloadType = 1
	TypeText PayloadType = 2
)

var (
	ErrHandshakeFailed  = errors.New("wormhole: handshake failed")
	ErrVerifyFailed     = errors.New("wormhole: verification failed (magic mismatch)")
	ErrInvalidFrameSize = errors.New("wormhole: invalid frame size")
)
