package wormhole

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"io"

	"github.com/schollz/pake/v3"
)

// PAKEHandshaker implements Handshaker using schollz/pake with Siec curve.
type PAKEHandshaker struct {
	Curve string
}

// NewPAKEHandshaker returns a default PAKE handshaker (curve: siec).
func NewPAKEHandshaker() *PAKEHandshaker {
	return &PAKEHandshaker{Curve: CurveSIEC}
}

// Run performs PAKE over the transport and returns the session key.
func (h *PAKEHandshaker) Run(transport FrameTransport, password string, isSender bool) ([]byte, error) {
	role := 0
	if !isSender {
		role = 1
	}
	p, err := pake.InitCurve([]byte(password), role, h.Curve)
	if err != nil {
		return nil, err
	}

	if isSender {
		// Alice: send A, read B, update
		if err := transport.SendFrame(p.Bytes()); err != nil {
			return nil, err
		}
		b, err := transport.ReadFrame()
		if err != nil {
			return nil, err
		}
		if err := p.Update(b); err != nil {
			return nil, err
		}
	} else {
		// Bob: read A, update, send B
		a, err := transport.ReadFrame()
		if err != nil {
			return nil, err
		}
		if err := p.Update(a); err != nil {
			return nil, err
		}
		if err := transport.SendFrame(p.Bytes()); err != nil {
			return nil, err
		}
	}

	key, err := p.SessionKey()
	if err != nil {
		return nil, err
	}
	return key, nil
}

// RunHandshake is a convenience that runs PAKE over io.ReadWriter.
// Uses default PAKEHandshaker and NewFrameTransport.
func RunHandshake(conn io.ReadWriter, password string, isSender bool) ([]byte, error) {
	transport := NewFrameTransport(conn)
	return NewPAKEHandshaker().Run(transport, password, isSender)
}

// AESCTRCipher implements StreamCipher using AES-256-CTR.
// IV is derived from SHA256 variants of the session key.
type AESCTRCipher struct{}

// NewDuplex returns enc/dec streams. Sender's enc and receiver's dec must use the same IV
// so decryption matches encryption. Role swap ensures: sender enc=iv1, receiver dec=iv1.
func (c *AESCTRCipher) NewDuplex(key []byte, isSender bool) (encStream, decStream cipher.Stream, err error) {
	digest := sha256.Sum256(key)
	aesKey := digest[:]

	iv1 := sha256.Sum256(append(key, 0x01))
	iv2 := sha256.Sum256(append(key, 0x02))

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, err
	}
	// Sender->receiver: sender enc=iv1, receiver dec=iv1.
	// Receiver->sender: receiver enc=iv2, sender dec=iv2.
	if isSender {
		return cipher.NewCTR(block, iv1[:16]), cipher.NewCTR(block, iv2[:16]), nil
	}
	return cipher.NewCTR(block, iv2[:16]), cipher.NewCTR(block, iv1[:16]), nil
}

// DefaultHandshaker is the default PAKE handshaker (injectable for tests).
var DefaultHandshaker Handshaker = NewPAKEHandshaker()

// DefaultStreamCipher is the default cipher (injectable for tests).
var DefaultStreamCipher StreamCipher = &AESCTRCipher{}
