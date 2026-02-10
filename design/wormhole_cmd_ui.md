```markdown
# Role
You are a Senior Go Engineer and System Architect. You are tasked with implementing a "Wormhole" feature for the `a-flex-box/cli` project. This feature allows secure, P2P-like file and text transfer between two CLI clients via a "dumb" relay server.

# Project Context
- **Framework**: Cobra (CLI), Viper (Config).
- **UI Library**: Lip Gloss (Styling), Bubble Tea / Bubbles (Progress bars).
- **Crypto**: `schollz/pake/v3` (Key Exchange), `crypto/aes` + `crypto/cipher` (Stream Encryption).
- **Philosophy**: "Smart Client, Dumb Server". The server is stateless and high-performance. The client handles all logic, encryption, and state.

# Objective
Implement the following modules:
1.  **Config Manager**: Persist relay aliases locally.
2.  **Relay Server**: A high-performance, concurrent TCP signal server.
3.  **Wormhole Core**: The crypto/protocol engine (PAKE + AES-CTR).
4.  **Client UI**: Send/Receive commands with dynamic progress bars.

---

# 1. Architecture & Directory Structure

Please create/update the following files:

```text
cmd/
  ├── relay.go         # [NEW] Server entry point (Systemd/Env friendly)
  ├── config.go        # [NEW] Config management (add/rm/list)
  ├── wormhole.go      # [NEW] Client entry point (send/receive)
internal/
  ├── config/          # [NEW] Viper wrapper
  │    └── manager.go
  ├── wormhole/        # [NEW] Core Logic
       ├── protocol.go # Header definitions, Payload types
       ├── crypto.go   # PAKE & AES wrapper
       ├── pool.go     # sync.Pool for zero-copy-ish buffers
       ├── server.go   # Relay logic
       ├── client.go   # Sender/Receiver logic
       └── ui.go       # Bubble Tea models

```

---

# 2. Detailed Implementation Specs

## Module A: Configuration (`internal/config`)

Use **Viper** to manage `~/.config/a-flex-box/config.yaml`.

* **Structure**:
```yaml
active_relay: "public"
relays:
  public: "tcp://relay.flex-box.dev:9000"
  local: "tcp://127.0.0.1:9000"

```


* **Commands (`cmd/config.go`)**:
* `cli config list`: Show table of relays (highlight active).
* `cli config use <name>`: Switch active relay.
* `cli config add <name> <addr>`: Add alias.
* `cli config rm <name>`: Remove alias.



## Module B: The Protocol (`internal/wormhole/protocol.go`)

Define a **Polymorphic Payload** system. The first encrypted frame after handshake MUST be a Header.

```go
type PayloadType uint8
const (
    TypeFile PayloadType = 1
    TypeText PayloadType = 2
)

type MetaHeader struct {
    Type     PayloadType `json:"t"`
    Name     string      `json:"n,omitempty"` // Filename (for files)
    Size     int64       `json:"s"`           // Total bytes
    Mode     uint32      `json:"m,omitempty"` // File permission (e.g. 0644)
}

```

## Module C: The "Dumb" Relay Server (`internal/wormhole/server.go`)

* **Performance**:
* Use `sync.Pool` to reuse `[]byte` buffers (32KB size) to reduce GC pressure.
* Use `SetNoDelay(true)` on TCP connections to minimize latency.


* **Logic**:
* Listen on a port.
* Read `RoomID` (first few bytes).
* Maintain a `map[string]net.Conn`.
* If match found: pipe `conn1 <-> conn2` using `io.CopyBuffer` (with pooled buffer).
* If no match: wait (with timeout).


* **Cmd Integration (`cmd/relay.go`)**:
* Bind flags to Environment Variables for Systemd/Docker support.
* Example: `--port` maps to `CLI_RELAY_PORT`.



## Module D: Crypto Engine (`internal/wormhole/crypto.go`)

* **Handshake**: Use `schollz/pake/v3` with curve `"siec"`.
* **Stream**:
* After PAKE, derive a Session Key.
* Wrap the `net.Conn` with `cipher.NewCTR` (AES-256).
* Ensure `IV` is handled (e.g., prepended or deterministic based on PAKE role).



## Module E: Client Logic & UI (`internal/wormhole/client.go` & `ui.go`)

* **Sender**:
* Connect to Relay -> Send PAKE ID -> Handshake -> Encrypt -> Send `MetaHeader` -> Send Body.
* If `TypeFile`: Use `bubbles/progress` to show upload bar.
* If `TypeText`: Just send string.


* **Receiver**:
* Connect to Relay -> Send PAKE ID -> Handshake -> Decrypt -> Read `MetaHeader`.
* If `TypeFile`: Show download bar, write to disk.
* If `TypeText`: Print to stdout (Lip Gloss styled box).


* **UI**:
* Use `lipgloss` for the summary box (e.g., "Secure Connection Established").
* Use `bubbles/progress` for the transfer progress.



---

# 3. Task Execution Steps

Please generate the code in the following order. **Do not hallucinate external dependencies other than the ones listed.**

1. **Step 1: Protocol & Pool**. Define `MetaHeader`, constants, and the `bufPool` in `internal/wormhole`.
2. **Step 2: Config Manager**. Implement `internal/config` and `cmd/config.go`.
3. **Step 3: Relay Server**. Implement `internal/wormhole/server.go` and `cmd/relay.go` (Focus on performance).
4. **Step 4: Crypto & Client Logic**. Implement PAKE handshake and AES wrapper in `crypto.go` and `client.go`.
5. **Step 5: UI Integration**. Implement `cmd/wormhole.go` connecting the logic with Bubbles UI.
6. **Step 6: Registration**. Show how to register these commands in `cmd/root.go`.

Start by generating Step 1 and Step 2.

```

```