#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 20:15:00
# original_prompt: 我现在需要你丰富一下makefile加上一个本地安装的命令,还有commit生成的prompt提示词的子命令,就是将这次的变更整体输出出来生成提示词,这个应该是作为prompt的子命令,makefile里面的安装还应该检测安装完成后时候可以直接使如果出现commond notfound这种错误应该输出
# summary: 增加 Makefile 本地安装检测与 Prompt Commit 变更感知功能
# action: 更新 Makefile 增加 install 目标并校验 PATH，更新 cmd/prompt.go 增加 commit 子命令以集成 git diff 输出
# expected_outcome: make install 可安装并提示 PATH 问题；bin/cli prompt commit "描述" 可生成包含代码变更的 AI 提示词
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}-> 正在升级 Makefile (增加 install 与 PATH 检测)...${NC}"

# ========================================================
# 1. 更新 Makefile
# ========================================================
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

# Colors (printf compatible)
CC_GREEN  := \033[0;32m
CC_CYAN   := \033[1;36m
CC_RED    := \033[0;31m
CC_YELLOW := \033[1;33m
CC_RESET  := \033[0m

.PHONY: all build clean test help register run install

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@printf "$(CC_CYAN)➜  Compiling...$(CC_RESET)\n"
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@printf "$(CC_GREEN)✔  Build ready: $(BUILD_DIR)/$(BINARY_NAME)$(CC_RESET)\n"

# install: 安装到 GOPATH/bin 并检查 PATH
install:
	@printf "$(CC_CYAN)➜  Installing to \$$(go env GOPATH)/bin ...$(CC_RESET)\n"
	@go install -ldflags "$(LDFLAGS)"
	@# 检测安装后的命令是否在 PATH 中可用
	@if command -v $(BINARY_NAME) >/dev/null 2>&1; then \
		printf "$(CC_GREEN)✔  Successfully installed!$(CC_RESET)\n"; \
		printf "   Location: $$(which $(BINARY_NAME))\n"; \
		printf "   You can now run '$(BINARY_NAME)' directly.\n"; \
	else \
		printf "$(CC_RED)✘  Installation successful, but '$(BINARY_NAME)' was NOT found in your PATH.$(CC_RESET)\n"; \
		printf "$(CC_YELLOW)⚠️  Please add the following to your shell profile (~/.bashrc or ~/.zshrc):$(CC_RESET)\n"; \
		printf "   export PATH=\$$PATH:$$(go env GOPATH)/bin\n"; \
	fi

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

echo -e "${GREEN}-> 正在升级 cmd/prompt.go (增加 commit 子命令)...${NC}"

# ========================================================
# 2. 更新 cmd/prompt.go
# ========================================================
cat << 'EOF' > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// 数据结构
type promptHistoryItem struct {
	Timestamp       string            `json:"timestamp"`
	OriginalPrompt  string            `json:"original_prompt"`
	Summary         string            `json:"summary"`
	Action          string            `json:"action"`
	ExpectedOutcome string            `json:"expected_outcome"`
	Context         map[string]string `json:"context,omitempty"`
}

var outputFormat string

// -----------------------------------------------------------
// 辅助函数: 生成基础 Context Prompt
// -----------------------------------------------------------
func buildContextPrompt() strings.Builder {
	historyPath := "history/history.json"
	var items []promptHistoryItem
	
	if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
		json.Unmarshal(data, &items)
	}

	var sb strings.Builder
	sb.WriteString("# Context: Project Development History\n")
	sb.WriteString("I am working on a Go CLI tool. Here is the summary of previous steps:\n\n")

	var lastStructure string

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("## Step %d (%s)\n", i+1, item.Timestamp))
		shortPrompt := strings.ReplaceAll(item.OriginalPrompt, "\n", " ")
		if len(shortPrompt) > 100 {
			shortPrompt = shortPrompt[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("- **Prompt Summary**: %s\n", shortPrompt)) 
		sb.WriteString(fmt.Sprintf("- **Action**: %s\n", item.Action))
		sb.WriteString(fmt.Sprintf("- **Outcome**: %s\n\n", item.ExpectedOutcome))

		if val, ok := item.Context["project_structure"]; ok && val != "" {
			lastStructure = val
		}
	}

	if lastStructure != "" {
		sb.WriteString("## Current Project Structure\n")
		sb.WriteString("```text\n")
		sb.WriteString(lastStructure)
		sb.WriteString("\n```\n\n")
	}
	
	sb.WriteString("--------------------------------------------------\n")
	return sb
}

// 辅助函数: 附加输出格式约束
func appendOutputConstraints(sb *strings.Builder, requirement string) {
	if outputFormat != "" {
		sb.WriteString(fmt.Sprintf("## Output Format Constraints\n"))
		sb.WriteString(fmt.Sprintf("1. Provide the solution as a **single %s file**.\n", outputFormat))
		sb.WriteString("2. **CRITICAL: METADATA HEADER REQUIRED**\n")
		sb.WriteString("   The file MUST start with a metadata header block in comments.\n")
		sb.WriteString("   The `original_prompt` field MUST contain the **EXACT FULL TEXT** of the 'New Requirement' section above. **DO NOT TRUNCATE, DO NOT SUMMARIZE.**\n\n")
		
		commentChar := "#"
		if outputFormat == "go" || outputFormat == "cpp" {
			commentChar = "//"
		}
		
		sb.WriteString("   Template:\n")
		sb.WriteString(fmt.Sprintf("   %s METADATA_START\n", commentChar))
		sb.WriteString(fmt.Sprintf("   %s timestamp: <YYYY-MM-DD HH:MM:SS>\n", commentChar))
		sb.WriteString(fmt.Sprintf("   %s original_prompt: %s\n", commentChar, requirement)) 
		sb.WriteString(fmt.Sprintf("   %s summary: <Short summary>\n", commentChar))
		sb.WriteString(fmt.Sprintf("   %s action: <Actions taken>\n", commentChar))
		sb.WriteString(fmt.Sprintf("   %s expected_outcome: <Outcome>\n", commentChar))
		sb.WriteString(fmt.Sprintf("   %s METADATA_END\n\n", commentChar))
	}
}

// -----------------------------------------------------------
// 主命令: prompt
// -----------------------------------------------------------
var promptCmd = &cobra.Command{
	Use:   "prompt [requirement]",
	Short: "生成 AI 提示词 (基础模式)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sb := buildContextPrompt()
		
		sb.WriteString("# New Requirement (Current Task)\n")
		sb.WriteString("Based on the context, please fulfill the following:\n\n")
		
		userRequirement := strings.Join(args, " ")
		sb.WriteString(userRequirement)
		sb.WriteString("\n\n")

		appendOutputConstraints(&sb, userRequirement)
		fmt.Println(sb.String())
	},
}

// -----------------------------------------------------------
// 子命令: prompt commit (带 Diff)
// -----------------------------------------------------------
var promptCommitCmd = &cobra.Command{
	Use:   "commit [instruction]",
	Short: "生成包含当前 Git 变更的 AI 提示词",
	Long:  `获取当前工作区所有未提交的变更 (git diff HEAD)，并将其加入到 Prompt 上下文中。适合让 AI 编写 Commit Message 或根据代码变更进行下一步开发。`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sb := buildContextPrompt()

		// 获取 Git Diff
		// git diff HEAD 会显示已暂存和未暂存的变更 (相对于最新的 commit)
		diffCmd := exec.Command("git", "diff", "HEAD")
		diffOut, err := diffCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error running git diff: %v\n", err)
			return
		}

		sb.WriteString("# Current Code Changes (Git Diff)\n")
		sb.WriteString("I have made the following changes to the codebase:\n\n")
		sb.WriteString("```diff\n")
		// 截断过长的 diff 避免 token 溢出 (可选，这里暂设 5000 字符)
		diffStr := string(diffOut)
		if len(diffStr) > 5000 {
			sb.WriteString(diffStr[:5000] + "\n... (diff truncated) ...")
		} else {
			sb.WriteString(diffStr)
		}
		sb.WriteString("\n```\n\n")

		sb.WriteString("--------------------------------------------------\n")
		sb.WriteString("# New Requirement / Instruction\n")
		sb.WriteString("Based on the history and the code changes above:\n\n")
		
		userRequirement := strings.Join(args, " ")
		sb.WriteString(userRequirement)
		sb.WriteString("\n\n")

		appendOutputConstraints(&sb, userRequirement)
		fmt.Println(sb.String())
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.AddCommand(promptCommitCmd) // 注册为子命令
	promptCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "shell", "Expected output format")
}
EOF

echo -e "${GREEN}-> 清理依赖并重新编译...${NC}"
go mod tidy
make build

echo -e "${GREEN}=== 升级完成！ ===${NC}"
echo -e "尝试本地安装检测: ${YELLOW}make install${NC}"
echo -e "尝试生成变更提示词: ${YELLOW}bin/cli prompt commit \"请帮我写一个 commit message\" -f shell${NC}"