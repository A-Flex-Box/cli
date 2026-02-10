🏗️ 架构重构：模块化 CMD 设计
我们采用以下规则来重构目录结构：

根命令保留：cmd/root.go 依然作为入口，但它不再包含具体的子命令逻辑。

子功能分包：每个大功能（如 wormhole, config）在 cmd/ 下拥有独立的目录。

工厂方法：子包不使用 init() 自动注册（这会导致副作用难以管理），而是导出一个 NewCommand() 函数，由 root.go 显式调用。

新的目录结构预览
📝 更新后的 Cursor 提示词 (Prompt)
我已将您的“分包”要求融入了这份最终的设计文档。请复制以下内容发给 Cursor。

2. Detailed Implementation Specs
Module A: CLI Wiring (cmd/)
cmd/root.go:

cmd/wormhole/root.go:

Define the parent wormhole command.

In NewCmd(), add newRelayCmd(), newSendCmd(), newReceiveCmd().

Module B: Configuration (internal/config)
Use Viper to manage ~/.config/a-flex-box/config.yaml.

Structure:

Commands (cmd/config/command.go):

list: Show table of relays.

use <name>: Switch active relay.

add <name> <addr>: Add alias.

Module C: The Protocol (internal/wormhole/protocol.go)
Define a Polymorphic Payload system.

Module D: The "Dumb" Relay Server (internal/wormhole/server.go)
Performance:

Use sync.Pool to reuse []byte buffers (32KB).

Use SetNoDelay(true).

Logic: Match RoomID. If match, io.CopyBuffer between conns.

Module E: Client Logic & UI (internal/wormhole/ui.go)
Bubble Tea Integration:

Create a custom tea.Model for the transfer progress.

Sender: Update model based on bytes written.

Receiver: Update model based on bytes read.

Use Lip Gloss to render a container box around the progress bar.

3. Task Execution Steps
Step 1: Protocol & Internal Logic

Implement internal/wormhole/protocol.go, pool.go, and crypto.go.

Implement internal/config/manager.go.

Step 2: Command Packages (cmd/)

Create cmd/config/command.go.

Create cmd/wormhole/ files. Ensure they export NewCmd().

Step 3: Relay Server Logic

Implement internal/wormhole/server.go.

Step 4: Client Logic & UI

Implement internal/wormhole/client.go and ui.go.

Step 5: Wiring

Modify cmd/root.go to integrate the new sub-packages.

Start by generating the code for Step 1 and Step 2.