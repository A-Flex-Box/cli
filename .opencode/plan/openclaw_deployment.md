# OpenClaw 部署功能实现计划

## 一、核心特性

| 特性 | 描述 |
|------|------|
| **双模式** | GUI 与 CMD 同时支持所有操作 |
| **组件选择器** | 安装时选择组件，自动处理依赖 |
| **EXE 加密** | Garble 混淆 + UPX 压缩 |
| **跨平台** | 支持 Windows / macOS / Linux |
| **环境探测** | 自动检测系统、依赖、服务状态 |

## 二、命令设计（GUI/CMD 双模式）

```bash
# 安装
cli openclaw install              # GUI 模式 (默认有显示环境)
cli openclaw install --no-gui     # CMD 模式 (无显示/脚本)
cli openclaw install --components "github,slack,notion"  # CMD 指定组件
cli openclaw install --method docker --version beta      # 指定方式和版本

# 卸载
cli openclaw uninstall            # GUI
cli openclaw uninstall --no-gui   # CMD
cli openclaw uninstall --purge    # 删除数据+配置

# 配置
cli openclaw config               # GUI 配置管理
cli openclaw config set key value # CLI 设置
cli openclaw config list          # CLI 列表

# 插件
cli openclaw plugins              # GUI 插件管理
cli openclaw plugins list         # CLI 列表
cli openclaw plugins install <name>

# 服务
cli openclaw start / stop / restart / status / logs

# 其他
cli openclaw update / doctor / version
```

## 三、组件依赖树设计

```go
type Component struct {
    ID           string
    Name         string
    Description  string
    Dependencies []string   // 依赖的其他组件ID
    Conflicts    []string   // 冲突组件
    Required     []string   // 必需的系统依赖
    Optional     bool       // 是否可选
    Category     string     // core, channel, tool, skill
}

// 组件定义
var Components = []Component{
    // === Core ===
    {ID: "core", Name: "OpenClaw Core", Category: "core", Optional: false},
    {ID: "gateway", Name: "Gateway Service", Category: "core", Dependencies: []string{"core"}},
    
    // === Channels ===
    {ID: "whatsapp", Name: "WhatsApp", Category: "channel", Dependencies: []string{"core"}},
    {ID: "telegram", Name: "Telegram", Category: "channel", Dependencies: []string{"core"}},
    {ID: "slack", Name: "Slack", Category: "channel", Dependencies: []string{"core"}},
    {ID: "discord", Name: "Discord", Category: "channel", Dependencies: []string{"core"}},
    {ID: "imessage", Name: "iMessage", Category: "channel", Dependencies: []string{"core"}, 
     Required: []string{"macos"}},
    
    // === Tools ===
    {ID: "browser", Name: "Browser Control", Category: "tool", Dependencies: []string{"core"}},
    {ID: "voice", Name: "Voice Call", Category: "tool", Dependencies: []string{"core", "audio"}},
    {ID: "canvas", Name: "Canvas", Category: "tool", Dependencies: []string{"browser"}},
    
    // === Skills ===
    {ID: "github", Name: "GitHub", Category: "skill", Dependencies: []string{"core"}},
    {ID: "notion", Name: "Notion", Category: "skill", Dependencies: []string{"core"}},
    {ID: "obsidian", Name: "Obsidian", Category: "skill", Dependencies: []string{"core"}},
    {ID: "spotify", Name: "Spotify Player", Category: "skill", Dependencies: []string{"core"}},
    {ID: "weather", Name: "Weather", Category: "skill", Dependencies: []string{"core"}},
    // ... 更多
}
```

## 四、目录结构

```
cmd/openclaw/
├── root.go
├── install.go
├── uninstall.go
├── config.go
├── plugins.go
├── service.go
├── update.go
├── doctor.go
└── version.go

internal/openclaw/
├── installer/
│   ├── installer.go       # Installer 接口
│   ├── detector.go        # 系统环境探测
│   ├── components.go      # 组件定义 + 依赖树
│   ├── resolver.go        # 依赖解析器
│   ├── native.go          # 原生安装实现
│   ├── docker.go          # Docker 安装实现
│   └── source.go          # 源码编译实现
├── platform/
│   ├── platform.go        # 平台抽象接口
│   ├── linux.go           # Linux (systemd)
│   ├── darwin.go          # macOS (launchd)
│   └── windows.go         # Windows (WSL2)
├── plugin/
│   ├── manager.go         # 插件管理器
│   ├── bundled.go         # 内置插件定义
│   └── clawhub.go         # ClawHub API
├── config/
│   ├── types.go           # 配置结构体
│   ├── generator.go       # 配置生成
│   └── validator.go       # 配置验证
├── service/
│   ├── manager.go         # 服务管理器
│   ├── systemd.go         # systemd 管理
│   ├── launchd.go         # launchd 管理
│   └── process.go         # 进程管理
├── action/                # GUI/CMD 双模式封装
│   ├── action.go          # Action 接口
│   ├── executor.go        # 执行器
│   └── progress.go        # 进度回调
└── gui/                   # Fyne GUI
    ├── app.go             # 主应用
    ├── theme.go           # 主题
    ├── screens/
    │   ├── install.go     # 安装向导
    │   ├── components.go  # 组件选择器
    │   ├── uninstall.go   # 卸载
    │   ├── config.go      # 配置页
    │   ├── plugins.go     # 插件页
    │   ├── status.go      # 状态页
    │   └── logs.go        # 日志页
    └── widgets/
        ├── card.go
        ├── component_list.go  # 组件列表组件
        ├── progress.go
        └── status_badge.go
```

## 五、GUI/CMD 双模式实现

```go
// internal/openclaw/action/action.go

type ProgressCallback func(stage string, current, total int, message string)

type Action interface {
    Execute(ctx context.Context, progress ProgressCallback) error
    Validate() error
    Steps() int
}

// GUI 使用: 提供进度回调更新 UI
// CMD 使用: 提供进度回调打印进度条
```

## 六、环境探测功能

```go
type SystemInfo struct {
    OS           string   // linux, darwin, windows
    Arch         string   // amd64, arm64
    Distro       string   // Ubuntu, macOS, etc.
    Version      string   // 22.04, 14.0, etc.

    // 依赖检测
    NodeVersion  string   // 检测 node --version
    NpmVersion   string   // 检测 npm --version
    DockerVer    string   // 检测 docker --version
    GitVersion   string   // 检测 git --version

    // 服务状态
    Systemd      bool     // Linux systemd 可用
    Launchd      bool     // macOS launchd 可用
    WSL          bool     // Windows WSL 环境

    // 网络
    PortsInUse   []int    // 占用的端口
    Tailscale    bool     // Tailscale 安装状态
}

func DetectSystem() *SystemInfo
func CheckDependencies(method InstallMethod) []DependencyError
```

## 七、平台支持策略

| 平台 | 安装方式 | 服务管理 |
|------|----------|----------|
| Linux | npm/curl + Docker | systemd |
| macOS | npm/curl + Homebrew | launchd |
| Windows | WSL2 (推荐) | WSL + systemd |

## 八、EXE 构建加密

**Makefile 添加**:

```makefile
# 构建加密版本
build-encrypted:
	garble -tiny -literals -seed=random build -ldflags="-s -w" -o bin/cli .
	upx --best --lzma bin/cli

# 构建所有平台
build-all:
	# Linux
	GOOS=linux GOARCH=amd64 garble -tiny -literals build -ldflags="-s -w" -o bin/cli-linux-amd64 .
	upx --best --lzma bin/cli-linux-amd64
	
	# macOS
	GOOS=darwin GOARCH=amd64 garble -tiny -literals build -ldflags="-s -w" -o bin/cli-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 garble -tiny -literals build -ldflags="-s -w" -o bin/cli-darwin-arm64 .
	
	# Windows
	GOOS=windows GOARCH=amd64 garble -tiny -literals build -ldflags="-s -w -H windowsgui" -o bin/cli-windows-amd64.exe .
	upx --best --lzma bin/cli-windows-amd64.exe
```

**加密工具依赖**:
- `garble` - Go 代码混淆
- `upx` - 可执行文件压缩

## 九、依赖添加

```go
// go.mod 新增
require (
    fyne.io/fyne/v2 v2.4.3
    github.com/cli/browser v1.3.0  // 打开浏览器
)

// 开发依赖
// go install mvdan.cc/garble@latest
// apt/brew install upx
```

## 十、实现阶段

| 阶段 | 内容 | 文件 |
|------|------|------|
| **Phase 1** | 基础框架 + 环境探测 + 组件定义 | `cmd/openclaw/root.go`, `internal/openclaw/installer/` |
| **Phase 2** | 安装/卸载核心逻辑 + 依赖解析 | `cmd/openclaw/install.go`, `cmd/openclaw/uninstall.go` |
| **Phase 3** | GUI 安装向导 + 组件选择器 | `internal/openclaw/gui/screens/install.go`, `components.go` |
| **Phase 4** | 配置管理 GUI + CMD | `cmd/openclaw/config.go`, `internal/openclaw/config/` |
| **Phase 5** | 插件管理 GUI + CMD | `cmd/openclaw/plugins.go`, `internal/openclaw/plugin/` |
| **Phase 6** | 服务管理 + 状态监控 GUI | `cmd/openclaw/service.go`, `internal/openclaw/service/` |
| **Phase 7** | EXE 加密构建 + 测试 | `Makefile` 更新 |

## 十一、GUI 界面设计

### 安装向导

```
┌─────────────────────────────────────────────────────────┐
│  🐾 OpenClaw Installer                                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Step 1: Installation Method                            │
│                                                         │
│  ○ Native (npm)              [Recommended]              │
│  ○ Docker                                              │
│  ○ Build from Source                                   │
│                                                         │
│  Version: [stable ▼]                                   │
│                                                         │
│  System Detected: Linux (Ubuntu 22.04)                 │
│  Node.js: ✓ 22.1.0    npm: ✓ 10.2.0                   │
│                                                         │
│              [ Cancel ]  [ Next → ]                     │
└─────────────────────────────────────────────────────────┘
```

### 组件选择器

```
┌─────────────────────────────────────────────────────────────────┐
│  📦 Component Selection                                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─ Core (Required) ──────────────────────────────────────────┐│
│  │ ☑ OpenClaw Core          Required component                ││
│  │ ☑ Gateway Service        HTTP API & WebSocket server       ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  ┌─ Channels ─────────────────────────────────────────────────┐│
│  │ ☑ Slack          Team messaging                  [2 deps] ││
│  │ ☑ Telegram       Bot integration                 [1 dep]  ││
│  │ ☐ Discord        Bot integration                 [1 dep]  ││
│  │ ☐ WhatsApp       Requires QR scan                [2 deps] ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  ┌─ Skills ───────────────────────────────────────────────────┐│
│  │ ☑ GitHub         Repository integration           [1 dep] ││
│  │ ☑ Notion         Workspace integration            [1 dep] ││
│  │ ☐ Obsidian       Notes integration                [1 dep] ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  Selected: 6 components    Dependencies: OK ✓                  │
│  Estimated disk: 245 MB    Download: 120 MB                    │
│                                                                 │
│              [ Back ]  [ Install ]                              │
└─────────────────────────────────────────────────────────────────┘
```

### 状态监控

```
┌─────────────────────────────────────────────────────────┐
│  📊 OpenClaw Status                              [● On] │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Gateway    ● Running    Uptime: 2h 34m                │
│  Port: 18789    PID: 12345    Memory: 128 MB           │
│                                                         │
│  Requests Today: 47    Total: 1,234                    │
│  Avg Response: 1.2s   Tokens: 45,678                   │
│                                                         │
│       [ Stop ]  [ Restart ]  [ View Logs ]             │
└─────────────────────────────────────────────────────────┘
```