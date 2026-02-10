您的观察非常准确且必要。

如果要彻底贯彻 **“模块化命令工厂”** 架构，**所有的顶层子命令**（包括现有的 `ai`, `history`, `doctor`, `archive` 等）都必须从 `cmd/` 根目录迁移到独立的子包中。

混合结构（一部分在根目录，一部分在子目录）是架构腐化的开始，会让代码库变得极其难以维护。

---

### 🏗️ 全局重构：Ultimate Modular Design

我们需要指示 Cursor 将现有的单文件命令重构为包结构。以下是修正后的完整目录蓝图：

```text
cmd/
├── root.go             # 纯净的入口，只负责 AddCommand(ai.NewCmd(), wormhole.NewCmd()...)
├── config/             # [NEW] 配置管理
│   └── command.go
├── wormhole/           # [NEW] 虫洞传输
│   ├── root.go
│   ├── relay.go
│   └── client.go
├── ai/                 # [REFACTOR] AI 助手
│   └── command.go      # 原 cmd/ai.go 迁移至此
├── history/            # [REFACTOR] 历史记录
│   ├── root.go         # history 父命令
│   └── add.go          # 原 cmd/history_add.go 迁移至此
├── doctor/             # [REFACTOR] 环境诊断
│   └── command.go      # 原 cmd/doctor.go 迁移至此
└── ... (archive, printer, prompt, validate 同理)

```

---

### 📝 最终版 Cursor 提示词 (Prompt)

这份提示词现在包含了 **“Refactoring Existing Commands” (重构现有命令)** 的明确指令。请复制以下内容发送给 Cursor。

---

```markdown
# Role
You are a Senior Go Engineer and System Architect.
Your goal is to **fully refactor** the `a-flex-box/cli` project into a modular "Command Factory" architecture, and then implement the new "Wormhole" feature.

# 1. Architectural Constraint: The "Clean Root" Policy
**STRICT RULE**: The `cmd/` directory must NOT contain any command logic files (like `ai.go`, `doctor.go`).
- **Input**: A flat `cmd/` directory with mixed files.
- **Output**: A structured `cmd/<package>/` layout.
- **Pattern**: Every subcommand is a package exporting `NewCmd(deps...)`.
- **Root**: `cmd/root.go` imports these packages and registers them.

# 2. Directory Structure Blueprint

Please enforce this structure. Create new directories/files and **move/refactor** existing code as needed.

```text
cmd/
  ├── root.go                  # Entry point. Registers ALL factories below.
  ├── config/                  # [NEW] Config Manager
  │    └── command.go          # func NewCmd(cfg *config.Root)
  ├── wormhole/                # [NEW] P2P Transfer
  │    ├── root.go             # func NewCmd(cfg *config.Wormhole)
  │    ├── relay.go            # subcommand: relay
  │    └── client.go           # subcommands: send, receive
  ├── ai/                      # [REFACTOR] Move cmd/ai.go here
  │    └── command.go          # func NewCmd(cfg *config.AI)
  ├── history/                 # [REFACTOR] Move cmd/history_add.go here
  │    ├── root.go             # func NewCmd() -> returns "history" parent cmd
  │    └── add.go              # subcommand: add
  ├── doctor/                  # [REFACTOR] Move cmd/doctor.go here
  │    └── command.go          # func NewCmd()
  └── ... (Apply same pattern to archive, printer, prompt, validate)

internal/
  ├── config/                  # [NEW] Viper Wrapper
  │    ├── types.go            # Nested structs (Root, Wormhole, AI...)
  │    └── manager.go          # Load() logic
  ├── wormhole/                # [NEW] Core Logic
       ├── protocol.go         # MetaHeader, PayloadType
       ├── pool.go             # sync.Pool
       ├── crypto.go           # PAKE + AES
       ├── server.go           # Relay Server
       ├── client.go           # Client Logic
       └── ui.go               # Bubble Tea UI

```

---

# 3. Refactoring Specifications

## A. Refactor Existing Commands (`ai`, `history`, `doctor`...)

For each existing file in `cmd/*.go` (except `root.go` and `main.go`):

1. **Move**: Create a directory `cmd/<name>/`.
2. **Package**: Change `package main` (or `cmd`) to `package <name>cmd`.
3. **Factory**: Wrap the global `var <Name>Cmd` into a function `func NewCmd(deps...) *cobra.Command`.
4. **Special Case (`history`)**:
* Create `cmd/history/root.go` for the parent `history` command.
* Refactor `cmd/history_add.go` into `cmd/history/add.go` and register it to the parent.



## B. Configuration Engine (`internal/config`)

Define a unified config struct to support dependency injection for the refactored commands.

```go
type Root struct {
    Wormhole WormholeConfig `mapstructure:"wormhole"`
    AI       AIConfig       `mapstructure:"ai"`
    // Add other modules as needed
}

```

## C. The Wormhole Feature (New Implementation)

* **Protocol**: Polymorphic Payload (TypeFile=1, TypeText=2).
* **Server**: Dumb Relay with `sync.Pool` (32KB buffers) and `TCP_NODELAY`.
* **Client**: PAKE Handshake -> AES-256-CTR Stream.
* **UI**: Bubble Tea Progress Bar in a Lip Gloss container.

---

# 4. Execution Plan

Please execute in this strict order to maintain build stability:

**Phase 1: Foundation & Config**

* Create `internal/config/`.
* Refactor `cmd/root.go` to support the factory pattern (but don't wire subcommands yet).

**Phase 2: Refactoring Existing Commands**

* Refactor `cmd/ai.go` -> `cmd/ai/`.
* Refactor `cmd/history_add.go` -> `cmd/history/`.
* Refactor `cmd/doctor.go` -> `cmd/doctor/`.
* (And others).
* Wire them back into `cmd/root.go`.

**Phase 3: Wormhole Implementation**

* Implement `internal/wormhole/` (Protocol, Crypto, Server, Client).
* Create `cmd/wormhole/` commands.
* Wire `wormhole` into `cmd/root.go`.

Start with **Phase 1 and Phase 2**.

```

```