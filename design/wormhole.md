📋 发送给 Cursor 的 Prompt
System / Context: You are an expert Golang engineer. I am working on a CLI tool (a-flex-box/cli). I need you to implement a new package internal/wormhole that handles secure P2P connection establishment using PAKE (Password-Authenticated Key Exchange).

Objective: Implement a secure "Wormhole" protocol that upgrades a raw net.Conn into an encrypted stream using a weak password.

Tech Stack:

PAKE Library: github.com/schollz/pake/v3 (Curve: "siec")

Encryption: crypto/aes, crypto/cipher (AES-256-CTR for stream encryption)

Protocol: Custom length-prefixed binary protocol.

1. Architecture Design (架构设计)
Please follow this file structure within internal/wormhole/:

Plaintext
internal/wormhole/
├── protocol.go      # Low-level framing (Read/Write length-prefixed bytes)
├── security.go      # PAKE handshake logic & AES Stream wrapper
├── session.go       # High-level Session struct (implements net.Conn)
└── types.go         # Error definitions and constants
2. Protocol Flow (协议流程)
The handshake happens immediately after a raw TCP connection is established.

Phase 1: Handshake (PAKE)

Sender (Alice) and Receiver (Bob) both initialize PAKE with the same password.

Sender sends PAKE public key A (prefixed with 4-byte length).

Receiver reads A, updates PAKE, and sends public key B (prefixed with 4-byte length).

Sender reads B, updates PAKE.

Both derive the shared SessionKey.

Phase 2: Verification (Ping-Pong)

Sender creates an AES-CTR stream using the SessionKey.

Sender encrypts and sends a magic string "WORMHOLE_OK".

Receiver decrypts the message. If it matches "WORMHOLE_OK", the channel is secure.

Phase 3: Transparent Tunnel

Return a wrapper object that implements io.ReadWriteCloser. Any data written to this wrapper is automatically encrypted; any data read is decrypted.

3. Implementation Details (代码规范)
A. protocol.go
Implement helper functions to handle message framing (prevent TCP sticking/segmentation).

Go
func sendFrame(w io.Writer, data []byte) error // Write uint32(len) + data
func readFrame(r io.Reader) ([]byte, error)    // Read uint32(len) + data
B. security.go
Implement the Handshake logic.

Go
// RunHandshake performs PAKE and returns the session key.
// isSender determines if we are Alice (id=0) or Bob (id=1).
func RunHandshake(conn io.ReadWriter, password string, isSender bool) ([]byte, error)
C. session.go (The Wrapper)
This is the most important part. I need a struct SecureConn that wraps the underlying net.Conn.

Go
type SecureConn struct {
    conn   net.Conn
    reader io.Reader // AES-CTR Reader
    writer io.Writer // AES-CTR Writer
}

// Upgrade takes a raw connection and a password, runs the handshake, 
// and returns a secured connection.
func Upgrade(conn net.Conn, password string, isSender bool) (*SecureConn, error) {
    // 1. RunHandshake
    // 2. Initialize AES-256-CTR reader/writer with the derived key
    // 3. Return SecureConn
}

// Read decrypts data on the fly
func (s *SecureConn) Read(p []byte) (n int, err error) {
    return s.reader.Read(p)
}

// Write encrypts data on the fly
func (s *SecureConn) Write(p []byte) (n int, err error) {
    return s.writer.Write(p)
}
4. Task Steps (执行步骤)
Please generate the code for internal/wormhole following these steps:

Write protocol.go for basic framing.

Write security.go implementing the schollz/pake/v3 logic we discussed (init curve "siec", exchange A/B).

Write session.go to implement the Upgrade function and the Read/Write wrappers using crypto/cipher.NewCTR. Crucial: Ensure the IV (Initialization Vector) for AES is handled correctly (usually derived from the handshake or prepended).

Write a simple usage example in a comment.