#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 18:45:00
# original_prompt: 现在丰富一下readme,然后需要你加个makefile以及go文件去实现刚刚的功能输出为shell...你还需要写一个validate来校验输出的文件...
# summary: 架构升级：实现基于文件的元数据感知、校验与自动归档
# action: 增加 internal/meta 解析包，新增 validate 和 history add 子命令，升级 Makefile 和 README
# expected_outcome: CLI 能自动识别代码文件头部的元数据，并通过 make register 将其录入历史记录
# METADATA_END

set -e

# ==========================================
# 0. 配置与环境检查
# ==========================================
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[1;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${CYAN}=== CLI 架构升级: 元数据感知与自动化归档 ===${NC}"

# 获取模块名 (从 go.mod 读取)
if [ ! -f go.mod ]; then
    echo -e "${RED}错误: 请在项目根目录下运行。${NC}"
    exit 1
fi
MODULE_NAME=$(grep "^module" go.mod | awk '{print $2}')

# ==========================================
# 1. 创建 internal/meta (核心解析逻辑)
# ==========================================
echo -e "${BLUE}-> [1/6] 创建元数据解析器 (internal/meta)...${NC}"
mkdir -p internal/meta

cat << EOF > internal/meta/parser.go
package meta

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// HistoryItem 对应 history.json 的结构
type HistoryItem struct {
	Timestamp       string \`json:"timestamp"\`
	OriginalPrompt  string \`json:"original_prompt"\`
	Summary         string \`json:"summary"\`
	Action          string \`json:"action"\`
	ExpectedOutcome string \`json:"expected_outcome"\`
}

// LanguageConfig 定义不同语言的注释风格
type LanguageConfig struct {
	CommentPrefix string
}

var langConfigs = map[string]LanguageConfig{
	"shell":  {"#"},
	"sh":     {"#"},
	"bash":   {"#"},
	"py":     {"#"},
	"python": {"#"},
	"yaml":   {"#"},
	"yml":    {"#"},
	"go":     {"//"},
	"cpp":    {"//"},
	"c":      {"//"},
	"java":   {"//"},
	"js":     {"//"},
	"ts":     {"//"},
	"sql":    {"--"},
}

// ParseMetadata 读取文件并提取 Metadata Block
func ParseMetadata(filePath string, lang string) (*HistoryItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 确定注释前缀
	config, ok := langConfigs[strings.ToLower(lang)]
	if !ok {
		// 默认 fallback
		config = LanguageConfig{CommentPrefix: "#"}
	}
	prefix := config.CommentPrefix

	scanner := bufio.NewScanner(file)
	inBlock := false
	item := &HistoryItem{}
	foundFields := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 检查开始标记
		if strings.Contains(line, "METADATA_START") {
			inBlock = true
			continue
		}
		// 检查结束标记
		if strings.Contains(line, "METADATA_END") {
			break
		}

		if inBlock {
			// 去除注释符号和空格
			content := strings.TrimPrefix(line, prefix)
			content = strings.TrimSpace(content)

			// 解析 Key: Value
			parts := strings.SplitN(content, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(strings.ToLower(parts[0]))
				val := strings.TrimSpace(parts[1])

				switch key {
				case "timestamp":
					item.Timestamp = val
					foundFields++
				case "original_prompt":
					item.OriginalPrompt = val
					foundFields++
				case "summary":
					item.Summary = val
					foundFields++
				case "action":
					item.Action = val
					foundFields++
				case "expected_outcome", "expected_result":
					item.ExpectedOutcome = val
					foundFields++
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 简单校验
	if foundFields < 3 {
		return nil, fmt.Errorf("metadata incomplete or missing (found %d fields). Ensure METADATA_START/END block exists with format 'key: value'", foundFields)
	}

	return item, nil
}
EOF

# ==========================================
# 2. 创建 cmd/validate.go
# ==========================================
echo -e "${BLUE}-> [2/6] 实现 validate 子命令...${NC}"

cat << EOF > cmd/validate.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"${MODULE_NAME}/internal/meta"

	"github.com/spf13/cobra"
)

var (
	isAnswer bool
	lang     string
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "校验文件格式或 AI 回答规范",
	Long:  \`校验指定文件。如果指定 --answer，将严格检查是否包含符合项目规范的历史元数据头。\`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		
		// 1. 基础文件存在性校验
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("❌ Error: File '%s' does not exist.\n", filePath)
			os.Exit(1)
		}

		fmt.Printf("🔍 Validating '%s'...\n", filePath)

		// 2. 如果是 Answer 模式，校验元数据
		if isAnswer {
			// 自动推断语言 (如果未指定)
			if lang == "" {
				ext := filepath.Ext(filePath)
				if len(ext) > 1 {
					lang = ext[1:] // remove dot
				} else {
					lang = "shell" // default
				}
			}

			item, err := meta.ParseMetadata(filePath, lang)
			if err != nil {
				fmt.Printf("❌ Metadata Validation Failed:\n   %v\n", err)
				fmt.Println("   Ensure the file contains a header like:")
				fmt.Println("   # METADATA_START")
				fmt.Println("   # timestamp: ...")
				fmt.Println("   # ...")
				fmt.Println("   # METADATA_END")
				os.Exit(1)
			}

			fmt.Println("✅ Metadata Header is Valid:")
			fmt.Printf("   - Timestamp: %s\n", item.Timestamp)
			fmt.Printf("   - Summary:   %s\n", item.Summary)
		}

		// 3. 这里可以预留接口做具体语言的 Syntax Check
		// 比如调用 go fmt 或 shfmt (如有)
		
		fmt.Println("✅ File validation passed.")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().BoolVar(&isAnswer, "answer", false, "Validate as an AI answer (require metadata)")
	validateCmd.Flags().StringVarP(&lang, "format", "f", "", "Source language (shell, go, python, etc.)")
}
EOF

# ==========================================
# 3. 创建 cmd/history_add.go
# ==========================================
echo -e "${BLUE}-> [3/6] 实现 history add 子命令...${NC}"

cat << EOF > cmd/history_add.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"${MODULE_NAME}/internal/meta"

	"github.com/spf13/cobra"
)

// historyCmd represents the history command base
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage project history",
}

var historyAddCmd = &cobra.Command{
	Use:   "add [file]",
	Short: "提取文件元数据并存入 history.json",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		
		// 1. 提取元数据
		// 默认认为 shell，或者根据后缀
		lang := "shell"
		if ext := filepath.Ext(filePath); len(ext) > 1 {
			lang = ext[1:]
		}
		
		newItem, err := meta.ParseMetadata(filePath, lang)
		if err != nil {
			fmt.Printf("❌ Failed to extract metadata: %v\n", err)
			os.Exit(1)
		}

		// 2. 读取现有 History
		historyPath := "history/history.json"
		var items []meta.HistoryItem

		// 检查文件是否存在
		if _, err := os.Stat(historyPath); err == nil {
			data, err := os.ReadFile(historyPath)
			if err != nil {
				fmt.Printf("❌ Error reading history file: %v\n", err)
				os.Exit(1)
			}
			// 只有文件不为空才解析
			if len(data) > 0 {
				if err := json.Unmarshal(data, &items); err != nil {
					// 尝试处理可能存在的尾部逗号等非标准 JSON (简单容错: 如果失败且非空，可能需要手动修复)
					fmt.Printf("⚠️  Warning: JSON parse error (might be malformed): %v. Initializing empty list.\n", err)
					items = []meta.HistoryItem{}
				}
			}
		}

		// 3. 追加并去重 (可选: 根据 timestamp 或 summary 去重，这里暂直接追加)
		items = append(items, *newItem)

		// 4. 回写
		newData, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			fmt.Printf("❌ Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(historyPath, newData, 0644); err != nil {
			fmt.Printf("❌ Error writing history file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully added history entry from '%s'.\n", filePath)
		fmt.Printf("   Summary: %s\n", newItem.Summary)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.AddCommand(historyAddCmd)
}
EOF

# ==========================================
# 4. 更新 cmd/prompt.go (支持 format 需求)
# ==========================================
echo -e "${BLUE}-> [4/6] 升级 prompt 命令 (添加格式指导)...${NC}"

cat << EOF > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"${MODULE_NAME}/internal/meta"

	"github.com/spf13/cobra"
)

var outputFormat string

var promptCmd = &cobra.Command{
	Use:   "prompt [requirement]",
	Short: "生成包含项目上下文的 AI 提示词",
	Long:  \`读取 history.json，结合当前需求，生成 Prompt。支持指定预期输出格式（如 shell, go 等），会自动附加元数据要求。\`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 读取 History
		historyPath := "history/history.json"
		var items []meta.HistoryItem
		
		if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
			json.Unmarshal(data, &items)
		}

		// 2. 构建 Prompt Context
		var sb strings.Builder
		sb.WriteString("# Context: Project Development History\n")
		sb.WriteString("I am working on a Go CLI tool. Here is the summary of previous development steps:\n\n")

		for i, item := range items {
			sb.WriteString(fmt.Sprintf("## Step %d (%s)\n", i+1, item.Timestamp))
			sb.WriteString(fmt.Sprintf("- **Original**: %s\n", item.OriginalPrompt))
			sb.WriteString(fmt.Sprintf("- **Summary**: %s\n", item.Summary))
			sb.WriteString(fmt.Sprintf("- **Action**: %s\n", item.Action))
			sb.WriteString(fmt.Sprintf("- **Outcome**: %s\n\n", item.ExpectedOutcome))
		}

		sb.WriteString("--------------------------------------------------\n")
		sb.WriteString("# New Requirement\n")
		sb.WriteString("Based on the context, please fulfill the following:\n\n")
		sb.WriteString(strings.Join(args, " "))
		sb.WriteString("\n\n")

		// 3. 添加格式与元数据约束
		if outputFormat != "" {
			sb.WriteString(fmt.Sprintf("## Output Format Constraints\n"))
			sb.WriteString(fmt.Sprintf("Please provide the solution as a **single %s file** (or script).\n", outputFormat))
			sb.WriteString("CRITICAL: The file MUST start with a metadata header block in comments. Format:\n")
			
			// 根据格式提示注释符号
			commentChar := "#"
			if outputFormat == "go" || outputFormat == "cpp" {
				commentChar = "//"
			}
			
			sb.WriteString(fmt.Sprintf("%s METADATA_START\n", commentChar))
			sb.WriteString(fmt.Sprintf("%s timestamp: <YYYY-MM-DD HH:MM:SS>\n", commentChar))
			sb.WriteString(fmt.Sprintf("%s original_prompt: <Your prompt here>\n", commentChar))
			sb.WriteString(fmt.Sprintf("%s summary: <Short summary>\n", commentChar))
			sb.WriteString(fmt.Sprintf("%s action: <What this script does>\n", commentChar))
			sb.WriteString(fmt.Sprintf("%s expected_outcome: <Expected result>\n", commentChar))
			sb.WriteString(fmt.Sprintf("%s METADATA_END\n\n", commentChar))
			sb.WriteString("Make sure the script is complete and executable.\n")
		}

		fmt.Println(sb.String())
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.Flags().StringVarP(&outputFormat, "format", "f", "shell", "Expected output format (shell, go, python, etc.)")
}
EOF

# ==========================================
# 5. 更新 README.md
# ==========================================
echo -e "${BLUE}-> [5/6] 刷新 README.md...${NC}"

cat << EOF > README.md
# ${MODULE_NAME} (Enhanced Edition)

![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Metadata](https://img.shields.io/badge/metadata-aware-blue)

这是一个具备**自我演进能力**的 Go CLI 工具。它不仅是一个构建工具，还自带了项目开发的历史记录管理功能。

## ✨ 核心特性

- **History Tracking**: \`cli history\` 命令集管理项目演进脉络。
- **Prompt Engineering**: \`cli prompt\` 自动生成包含上下文的 AI 提示词。
- **Metadata Aware**: 能够识别代码文件头部包含的结构化元数据（Prompt/Summary/Action）。
- **Format Validation**: \`cli validate\` 校验 AI 输出是否符合工程规范。

## 🚀 快速开始

### 1. 生成需求 Prompt
告诉 AI 你想要什么，并指定输出格式（比如 shell）：

\`\`\`bash
make run ARGS='prompt "添加一个新功能" -f shell'
# 复制输出内容发送给 AI
\`\`\`

### 2. 接收并注册 AI 的回答
将 AI 生成的带元数据的脚本保存为 \`ai_response.sh\`，然后执行：

\`\`\`bash
# 自动校验元数据格式，并录入 history.json，最后移动到归档目录
make register FILE=ai_response.sh
\`\`\`

## 📂 目录结构

- \`cmd/\`: Cobra 命令定义
- \`internal/meta/\`: 元数据解析核心逻辑
- \`history/shell/\`: 归档的历史操作脚本
- \`history/history.json/\`: 结构化的项目演进数据库

## 🛠 开发指令

\`\`\`bash
make build       # 编译
make test        # 测试
make register    # 注册脚本到历史
\`\`\`
EOF

# ==========================================
# 6. 更新 Makefile (增加 register 目标)
# ==========================================
echo -e "${BLUE}-> [6/6] 升级 Makefile...${NC}"

cat << 'EOF' > Makefile
# ---------------------------------------------------------
#  Config
# ---------------------------------------------------------
BINARY_NAME := cli
BUILD_DIR   := bin
HISTORY_DIR := history/shell

# Git Info
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE        := $(shell date +%Y-%m-%dT%H:%M:%S%z)

LDFLAGS     := -X '$(shell go list -m)/cmd.version=$(VERSION)' \
               -X '$(shell go list -m)/cmd.commit=$(COMMIT)' \
               -X '$(shell go list -m)/cmd.date=$(DATE)' \
               -s -w

# Colors
CC_GREEN  := \033[0;32m
CC_CYAN   := \033[1;36m
CC_RED    := \033[0;31m
CC_RESET  := \033[0m

.PHONY: all build clean test help register run

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@printf "$(CC_CYAN)➜  Compiling...$(CC_RESET)\n"
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@printf "$(CC_GREEN)✔  Build ready: $(BUILD_DIR)/$(BINARY_NAME)$(CC_RESET)\n"

# register: 校验 -> 录入历史 -> 归档文件
# 用法: make register FILE=xxx.sh
register: build
	@if [ -z "$(FILE)" ]; then \
		printf "$(CC_RED)Error: FILE argument is missing. Usage: make register FILE=script.sh$(CC_RESET)\n"; \
		exit 1; \
	fi
	@printf "$(CC_CYAN)➜  Validating Metadata in $(FILE)...$(CC_RESET)\n"
	@$(BUILD_DIR)/$(BINARY_NAME) validate --answer $(FILE)
	@printf "$(CC_CYAN)➜  Adding to Project History...$(CC_RESET)\n"
	@$(BUILD_DIR)/$(BINARY_NAME) history add $(FILE)
	@printf "$(CC_CYAN)➜  Archiving file...$(CC_RESET)\n"
	@mkdir -p $(HISTORY_DIR)
	@TS=$$(date +%Y%m%d_%H%M%S); \
	mv $(FILE) $(HISTORY_DIR)/$${TS}_$$(basename $(FILE)); \
	printf "$(CC_GREEN)✔  Registered & Archived to $(HISTORY_DIR)/$${TS}_$$(basename $(FILE))$(CC_RESET)\n"

run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

clean:
	@rm -rf $(BUILD_DIR)

test:
	@go test -v ./...
EOF

# ==========================================
# 收尾
# ==========================================

# 重新编译以确保新功能可用
echo -e "${YELLOW}正在编译新版 CLI...${NC}"
go mod tidy
make build

echo -e "\n${GREEN}=== 升级完成！ ===${NC}"
echo -e "你可以立即使用以下命令测试本次升级脚本本身（因为它包含了元数据头）："
echo -e "${CYAN}make register FILE=upgrade_cli_metadata.sh${NC}"
echo -e "\n然后尝试生成下一个需求的提示词："
echo -e "${CYAN}bin/cli prompt \"帮我加一个查看系统信息的子命令\" -f go${NC}"