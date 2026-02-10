这是一份完整的、经过深思熟虑的架构设计方案与代码实现提示词（Prompt）。这份文档采用了 **COAI (Context-Objective-Architecture-Implementation)** 标准，专为 Cursor 设计，旨在让 AI 一次性生成高质量的、模块化的工业级代码。

请直接复制以下 **Markdown 内容** 发送给 Cursor。

---

```markdown
# Role
You are a Senior Go Engineer and System Architect. You are tasked with refactoring the `a-flex-box/cli` project to support a modular CLI structure and implementing the "Wormhole" (P2P-like transfer) feature.

# Architectural Principles
1.  **Command Factory Pattern**: All sub-commands must reside in their own packages under `cmd/` (e.g., `cmd/wormhole`, `cmd/config`) and export a `NewCmd(deps...)` function.
2.  **Dependency Injection**: Configuration should be loaded in `root.go` and injected into sub-commands. Do not use global Viper instances inside sub-commands.
3.  **Smart Client, Dumb Server**: The Relay server is stateless and handles raw TCP streams. All logic (encryption, UI, state) resides in the client.
4.  **Performance First**: Use `sync.Pool` for zero-copy buffers and `TCP_NODELAY`.

# Project Context
- **Frameworks**: Cobra (CLI), Viper (Config).
- **UI**: Lip Gloss (Styling), Bubble Tea & Bubbles (Progress Bars).
- **Crypto**: `schollz/pake/v3` (Handshake), `crypto/aes` + `crypto/cipher` (Stream).

---

# 1. Directory Structure Blueprint

Please adhere strictly to this file structure:

```text
cmd/
  ├── root.go                  # Main entry, loads config, registers factories
  ├── config/                  # [NEW PACKAGE]
  │    └── command.go          # func NewCmd(cfg *config.Root)
  ├── wormhole/                # [NEW PACKAGE]
       ├── root.go             # func NewCmd(cfg *config.Wormhole)
       ├── relay.go            # subcommand: relay
       └── client.go           # subcommands: send, receive
internal/
  ├── config/                  # Configuration Logic
  │    ├── types.go            # Struct definitions
  │    └── manager.go          # Viper loader
  ├── wormhole/                # Core Business Logic
       ├── protocol.go         # Header definitions, Payload types
       ├── pool.go             # sync.Pool for high-perf buffers
       ├── crypto.go           # PAKE & AES wrapper
       ├── server.go           # Relay server logic
       ├── client.go           # Sender/Receiver logic
       └── ui.go               # Bubble Tea models

```

---

# 2. Detailed Implementation Specifications

## Step 1: Configuration Engine (`internal/config`)

**`internal/config/types.go`**:
Define a nested configuration structure to allow modular injection.

```go
package config

type Root struct {
    Debug    bool           `mapstructure:"debug" yaml:"debug"`
    Wormhole WormholeConfig `mapstructure:"wormhole" yaml:"wormhole"`
}

type WormholeConfig struct {
    ActiveRelay string            `mapstructure:"active_relay" yaml:"active_relay"`
    Relays      map[string]string `mapstructure:"relays"       yaml:"relays"`
}

```

**`internal/config/manager.go`**:
Implement `Load()` which reads `~/.config/a-flex-box/config.yaml` using Viper and unmarshals it into `&Root{}`.

## Step 2: The Command Factories (`cmd/`)

**`cmd/config/command.go`**:

* `NewCmd(cfg *config.Root)`: Returns the config management command tree.
* Implement `list` (use Lipgloss table), `use`, `add`, `rm`.

**`cmd/wormhole/root.go`**:

* `NewCmd(cfg *config.WormholeConfig)`: Returns the parent `wormhole` command.
* Registers `newRelayCmd()` and `newClientCmds(cfg)`.

## Step 3: Wormhole Protocol & Core (`internal/wormhole`)

**`protocol.go`**:
Define the "Polymorphic Payload" header.

```go
type PayloadType uint8
const (
    TypeFile PayloadType = 1
    TypeText PayloadType = 2
)
type MetaHeader struct {
    Type PayloadType `json:"t"`
    Name string      `json:"n,omitempty"`
    Size int64       `json:"s"`
    Mode uint32      `json:"m,omitempty"` // File permissions
}

```

**`pool.go`**:
Implement a global `sync.Pool` that returns `*[]byte` (size 32KB) to minimize GC pressure during transfer.

**`crypto.go`**:

* `RunHandshake(conn, password)`: Uses `pake/v3`.
* `NewSecureConn(conn, key)`: Wraps connection with `cipher.NewCTR` (AES-256).

## Step 4: The Dumb Relay (`internal/wormhole/server.go`)

This must be highly optimized.

* **Logic**:
1. Accept TCP connection.
2. Read first N bytes (RoomID).
3. Store in `map[string]net.Conn`.
4. If match found: Use `io.CopyBuffer` with buffers from `pool.go` to pipe data between two connections.
5. Close connections when done.


* **System**: Set `SetNoDelay(true)` on connections.

## Step 5: Client UI & Logic (`internal/wormhole/client.go` & `ui.go`)

**`client.go`**:

* **Send**: Connect -> PAKE -> Encrypt -> Send `MetaHeader` -> Send Body.
* **Receive**: Connect -> PAKE -> Decrypt -> Read `MetaHeader` -> Read Body.

**`ui.go`**:

* Create a `tea.Model` using `bubbles/progress`.
* Wrap the progress bar in a Lip Gloss container (border, padding).
* Update the model via a custom `ProgressMsg` sent from the transfer loop.

---

# 3. Execution Plan

Please generate the code in the following phases. **Pause after each phase for review.**

### Phase 1: Foundation & Config

* Create `internal/config/types.go` and `manager.go`.
* Create `cmd/config/command.go`.
* Create `internal/wormhole/protocol.go` and `pool.go`.

### Phase 2: Relay Server

* Create `internal/wormhole/server.go`.
* Create `cmd/wormhole/relay.go`.

### Phase 3: Crypto & Core Logic

* Create `internal/wormhole/crypto.go`.
* Create `internal/wormhole/client.go` (Logic only, no UI yet).

### Phase 4: UI & Wiring

* Create `internal/wormhole/ui.go` (Bubble Tea integration).
* Update `internal/wormhole/client.go` to use UI.
* Create `cmd/wormhole/root.go` and `client.go`.
* Update `cmd/root.go` to wire everything together.

Start with **Phase 1**.

```

```