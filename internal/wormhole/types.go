package wormhole

import (
	"crypto/cipher"
	"errors"
)

// Magic string sent after handshake to verify the channel is secure.
const MagicVerify = "WORMHOLE_OK"

// Curve name for PAKE (siec is fast and secure).
const CurveSIEC = "siec"

var (
	ErrHandshakeFailed  = errors.New("wormhole: handshake failed")
	ErrVerifyFailed     = errors.New("wormhole: verification failed (magic mismatch)")
	ErrInvalidFrameSize = errors.New("wormhole: invalid frame size")
)

// FrameTransport handles length-prefixed message framing.
// Implementations can vary (TCP, UDP with reassembly, etc.).
type FrameTransport interface {
	SendFrame(data []byte) error
	ReadFrame() ([]byte, error)
}

// Handshaker performs PAKE key exchange and returns the session key.
// Different implementations can use different curves or protocols.
type Handshaker interface {
	// Run performs the handshake over the given transport.
	// isSender: true for Alice (id=0), false for Bob (id=1).
	Run(transport FrameTransport, password string, isSender bool) (sessionKey []byte, err error)
}

// StreamCipher creates encrypt/decrypt streams from a session key.
// Allows swapping AES-CTR for other ciphers (e.g. ChaCha20).
type StreamCipher interface {
	// NewDuplex returns two cipher.Stream instances for duplex: enc for our writes, dec for our reads.
	// They must use different IVs/nonces to avoid keystream reuse.
	NewDuplex(key []byte) (encStream, decStream cipher.Stream, err error)
}
