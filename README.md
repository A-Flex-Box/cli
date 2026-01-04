# github.com/A-Flex-Box/cli (Enhanced Edition)

![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Metadata](https://img.shields.io/badge/metadata-aware-blue)

这是一个具备**自我演进能力**的 Go CLI 工具。它不仅是一个构建工具，还自带了项目开发的历史记录管理功能。

## ✨ 核心特性

- **History Tracking**: `cli history` 命令集管理项目演进脉络。
- **Prompt Engineering**: `cli prompt` 自动生成包含上下文的 AI 提示词。
- **Metadata Aware**: 能够识别代码文件头部包含的结构化元数据（Prompt/Summary/Action）。
- **Format Validation**: `cli validate` 校验 AI 输出是否符合工程规范。

## 🚀 快速开始

### 1. 生成需求 Prompt
告诉 AI 你想要什么，并指定输出格式（比如 shell）：

```bash
make run ARGS='prompt "添加一个新功能" -f shell'
# 复制输出内容发送给 AI
```

### 2. 接收并注册 AI 的回答
将 AI 生成的带元数据的脚本保存为 `ai_response.sh`，然后执行：

```bash
# 自动校验元数据格式，并录入 history.json，最后移动到归档目录
make register FILE=ai_response.sh
```

## 📂 目录结构

- `cmd/`: Cobra 命令定义
- `internal/meta/`: 元数据解析核心逻辑
- `history/shell/`: 归档的历史操作脚本
- `history/history.json/`: 结构化的项目演进数据库

## 🛠 开发指令

```bash
make build       # 编译
make test        # 测试
make register    # 注册脚本到历史
```
