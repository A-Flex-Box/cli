# github.com/A-Flex-Box/cli (Enhanced Edition)

![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Metadata](https://img.shields.io/badge/metadata-aware-blue)

这是一个具备**自我演进能力**的 Go CLI 工具。它不仅是一个构建工具，还自带了项目开发的历史记录管理功能。

## ✨ 核心特性

- **History Tracking**: `cli history` 命令集管理项目演进脉络
- **Prompt Engineering**: `cli prompt` 自动生成包含上下文的 AI 提示词
- **Metadata Aware**: 能够识别代码文件头部包含的结构化元数据（Prompt/Summary/Action）
- **Format Validation**: `cli validate` 校验 AI 输出是否符合工程规范
- **Printer Management**: `cli printer` 打印机和扫描仪管理工具
- **Archive Management**: `cli archive` 创建 tar.gz 归档文件
- **AI Tools**: `cli ai` AI 工程化辅助工具
- **Environment Check**: `cli doctor` 检查环境健康状态

## 🚀 快速开始

### 安装

```bash
go install github.com/A-Flex-Box/cli@latest
```

### 基本使用

```bash
# 查看所有可用命令
cli --help

# 查看特定命令的帮助
cli printer --help
```

## 📂 目录结构

```
.
├── cmd/              # Cobra 命令定义
│   ├── archive.go   # 归档命令
│   ├── ai.go        # AI工具命令
│   ├── doctor.go    # 环境检查命令
│   ├── history_add.go # 历史记录添加命令
│   ├── printer.go   # 打印机管理命令
│   ├── prompt.go    # Prompt生成命令
│   ├── root.go      # 根命令
│   └── validate.go  # 验证命令
├── internal/        # 内部包
│   ├── archiver/    # 归档逻辑
│   ├── fsutil/      # 文件系统工具
│   ├── logger/      # 日志工具
│   ├── meta/        # 元数据解析
│   └── printer/     # 打印机功能
├── history/         # 历史记录
│   ├── shell/       # 归档的历史操作脚本
│   ├── history.json # 结构化的项目演进数据库
│   └── history.md    # 人类可读的历史记录
├── Makefile         # 构建脚本
└── README.md        # 本文档
```

## 🛠 开发指令

```bash
make build       # 编译
make test        # 测试
make register    # 注册脚本到历史
make help        # 查看所有可用命令
```

---

## 📋 命令详细说明

### 1. `cli history add` - 历史记录管理

将带元数据的文件添加到项目历史记录中。

**用法：**
```bash
cli history add <file>
```

**示例：**
```bash
# 添加一个shell脚本到历史记录
cli history add ai_response.sh
```

**功能：**
- 自动提取文件头部的元数据（timestamp, summary, action等）
- 生成项目结构快照
- 将记录追加到 `history/history.json`
- 文件移动到 `history/shell/` 目录

---

### 2. `cli prompt` - AI 提示词生成

生成包含项目上下文的 AI 提示词，用于与 AI 交互。

**用法：**
```bash
cli prompt [requirement] [flags]
```

**选项：**
- `-f, --format`: 期望的输出格式 (shell, go, python, sql等)

**示例：**
```bash
# 生成基础提示词
cli prompt "添加一个新功能"

# 指定输出格式为shell脚本
cli prompt "优化代码结构" -f shell

# 生成Go代码提示词
cli prompt "实现用户认证" -f go
```

**子命令：**
- `cli prompt commit [instruction]`: 根据 Git 变更生成 Commit Message 提示词

**示例：**
```bash
# 生成commit message提示词
cli prompt commit

# 带自定义指令
cli prompt commit "重点关注性能优化"
```

---

### 3. `cli validate` - 文件验证

校验文件格式或 AI 回答规范。

**用法：**
```bash
cli validate <file> [flags]
```

**选项：**
- `--answer`: 验证为 AI 回答（要求包含元数据）
- `-f, --format`: 源语言 (shell, go, python等)

**示例：**
```bash
# 基础文件存在性校验
cli validate script.sh

# 验证AI回答的元数据格式
cli validate ai_response.sh --answer

# 指定语言格式验证
cli validate solution.go --answer -f go
```

---

### 4. `cli printer` - 打印机和扫描仪管理

打印机和扫描仪管理工具，支持自动发现、打印和扫描功能。

#### 4.1 自动发现和配置打印机

```bash
# 自动扫描网络打印机并添加到CUPS
cli printer --setup
```

#### 4.2 打印PDF文件

**本地文件打印：**
```bash
# 自动选择第一台打印机
cli printer --file document.pdf --auto

# 指定打印机名称
cli printer --file document.pdf --printer "EPSON_EM-C8101_Series"

# 交互式选择打印机
cli printer --file document.pdf

# 指定打印选项（2份，单面，彩色）
cli printer --file document.pdf --copies 2 --sides one-sided --color color --cups

# 双面打印，黑白
cli printer --file document.pdf --sides two-sided-long-edge --color monochrome --cups
```

**远程URL打印（自动下载到临时目录）：**
```bash
# 从URL下载并打印（自动清理临时文件）
cli printer --url "https://example.com/document.pdf" --auto

# 指定打印机和选项
cli printer --url "https://example.com/document.pdf" --printer "MyPrinter" --copies 2 --cups
```

**打印选项：**
- `--copies`: 打印份数 (1-999)
- `--sides`: 单双面设置
  - `one-sided`: 单面
  - `two-sided-long-edge`: 双面长边翻转
  - `two-sided-short-edge`: 双面短边翻转
- `--color`: 颜色模式
  - `auto`: 自动
  - `color`: 彩色
  - `monochrome`: 黑白
- `--source`: 纸张来源
  - `auto`: 自动
  - `manual`: 手动进纸
  - `adf`: 自动文档进纸器
  - `tray-1`, `tray-2`: 纸盒1/2
- `--cups`: 使用CUPS lp命令（推荐，支持所有选项）

#### 4.3 扫描文档

**列出可用扫描设备：**
```bash
cli printer --list-scan-devices
```

**基本扫描：**
```bash
# 自动选择设备扫描
cli printer --scan

# 指定扫描设备（airscan设备）
cli printer --scan --scan-device "airscan:w0:EPSON EM-C8101 Series"

# 平板扫描
cli printer --scan --scan-source flatbed --scan-format pdf

# ADF批量扫描多页（自动扫描所有页面）
cli printer --scan --scan-source adf --scan-format jpeg --scan-batch

# 指定扫描选项（600 DPI，灰度，ADF）
cli printer --scan --scan-source adf --scan-resolution 600 --scan-color grayscale
```

**扫描选项：**
- `--scan-device`: 扫描设备名称
- `--scan-output`: 输出文件路径（默认自动生成）
- `--scan-resolution`: 分辨率DPI (150, 200, 300, 600)
- `--scan-color`: 颜色模式 (color, grayscale, lineart)
- `--scan-source`: 扫描源 (flatbed, adf)
- `--scan-format`: 输出格式 (pdf, jpeg, png)
- `--scan-batch`: 批量扫描模式（ADF多页）
- `--scan-batch-format`: 批量扫描文件名格式（如 scan_%03d.jpg）

**完整示例：**
```bash
# 打印选项说明
cli printer --file doc.pdf \
  --printer "EPSON_EM-C8101_Series" \
  --copies 2 \
  --sides two-sided-long-edge \
  --color color \
  --source tray-1 \
  --cups

# ADF批量扫描并保存为PDF
cli printer --scan \
  --scan-device "airscan:w0:EPSON EM-C8101 Series" \
  --scan-source adf \
  --scan-format pdf \
  --scan-resolution 300 \
  --scan-color color

# 远程URL打印
cli printer --url "https://example.com/report.pdf" \
  --auto \
  --copies 1 \
  --sides one-sided \
  --color auto \
  --cups
```

---

### 5. `cli archive` - 归档管理

创建 tar.gz 归档文件，可选择是否删除源文件。

**用法：**
```bash
cli archive [flags]
```

**选项：**
- `-d, --delete`: 归档后删除源文件

**示例：**
```bash
# 创建归档（保留源文件）
cli archive

# 创建归档并删除源文件
cli archive --delete
```

**功能：**
- 自动生成带时间戳的归档文件名（格式：`archive_YYYYMMDD_HHMMSS.tar.gz`）
- 排除 `.git`、`history`、历史归档文件等
- 保留历史标准归档文件

---

### 6. `cli ai` - AI 工程化辅助工具

AI 工程化辅助工具，包含环境检查和项目初始化功能。

#### 6.1 `cli ai setup` - 环境自检

检查 GPU、CUDA 及虚拟环境列表。

**用法：**
```bash
cli ai setup
```

**功能：**
- 检查 NVIDIA 驱动和 GPU 信息
- 检查 PyTorch 和 CUDA 环境
- 列出所有 conda 虚拟环境
- 显示当前活动的虚拟环境

**示例：**
```bash
cli ai setup
```

#### 6.2 `cli ai init` - 项目初始化

生成 AI 项目标准目录结构。

**用法：**
```bash
cli ai init <project_name>
```

**示例：**
```bash
# 创建名为 myproject 的AI项目
cli ai init myproject
```

**生成的目录结构：**
```
myproject/
├── data/
│   ├── raw/        # 原始不可变数据
│   └── processed/  # 清洗后的特征数据
├── models/         # 模型权重 checkpoints
├── notebooks/      # Jupyter Notebooks
├── src/            # 源代码
│   └── utils/      # 工具函数
├── logs/           # Training Logs
└── configs/        # Hyperparameters
```

---

### 7. `cli doctor` - 环境健康检查

检查环境健康状态，验证 Git、Go 以及项目配置文件是否存在且正常。

**用法：**
```bash
cli doctor
```

**功能：**
- 检查 Go 是否已安装
- 检查 Git 是否已安装
- 检查 Make 是否已安装
- 检查历史数据库是否存在

**示例：**
```bash
cli doctor
```

---

## 📝 注意事项

### Printer 命令注意事项

1. **远程URL打印**: 文件会自动下载到系统临时目录（`/tmp/printer_downloads`），打印完成后自动清理
2. **CUPS模式**: 使用 `--cups` 选项可以获得更好的打印选项支持（颜色、纸张来源等）
3. **扫描设备**: 程序会自动过滤摄像头设备，优先选择打印机扫描设备
4. **批量扫描**: ADF扫描时，如果输出格式不是PDF，会自动启用批量扫描模式

### 通用注意事项

1. **日志系统**: 所有命令使用统一的 `zap` 日志库，提供结构化日志输出
2. **历史记录**: 使用 `cli history add` 命令可以将带元数据的文件添加到项目历史记录
3. **元数据格式**: AI 生成的代码文件应包含元数据头部，格式参考 `internal/meta/parser.go`

---

## 🔧 开发

### 构建

```bash
make build
```

### 测试

```bash
make test
```

### 运行

```bash
make run ARGS="<command> <args>"
```

---

## 开发规范

### 枚举规范

1. **类型化枚举**：所有枚举值应使用类型别名定义，禁止使用裸字符串。

   ```go
   type EnumName string

   const (
       EnumValue1 EnumName = "value1"
       EnumValue2 EnumName = "value2"
       EnumNone   EnumName = "none"   // 表示空/无/零值语义
   )
   ```

2. **禁止空字符串与零值**：枚举不得使用空字符串 `""` 或类似零值。表示「无」「不适用」等语义时，应使用显式值如 `"none"`、`"na"` 等。

3. **命名约定**：常量名采用 `类型名 + 用途` 的 PascalCase，如 `InstallStatusInstalled`、`PortStatusNone`。

4. **注册与使用**：在包内通过 `const` 块统一声明，使用时通过常量引用，避免魔法字符串。

---

## 📄 License

See LICENSE file for details.
