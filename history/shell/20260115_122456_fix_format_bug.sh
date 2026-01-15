#!/bin/bash
# METADATA_START
# timestamp: 2026-01-05 15:00:00
# original_prompt: (Output with %!(EXTRA...) bugs)
# summary: 修复 cmd/prompt.go 中 fmt.Sprintf 占位符错误的 Bug
# action: 重写 cmd/prompt.go。使用分段写入策略：Import 部分允许变量扩展，逻辑部分完全禁用 Shell 扩展，将 %%d 修正为 %d，确保 Go 格式化字符串正确。
# expected_outcome: bin/cli prompt 输出正常的格式化文本，不再出现 %!(EXTRA...) 错误。
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# 自动获取模块名
if [ ! -f go.mod ]; then
    echo "❌ go.mod not found!"
    exit 1
fi
MODULE_NAME=$(grep "^module" go.mod | awk '{print $2}')

echo -e "${GREEN}-> 正在修复 cmd/prompt.go (修正 fmt.Sprintf 格式符)...${NC}"

# ==============================================================================
# 第一部分：写入头部 (使用 EOF，允许 ${MODULE_NAME} 扩展)
# ==============================================================================
cat << HEADER_EOF > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"${MODULE_NAME}/internal/fsutil"
)
HEADER_EOF

# ==============================================================================
# 第二部分：写入逻辑 (使用 'BODY_EOF'，彻底禁用 Shell 扩展)
# 关键修复：这里的 %d, %s 不会被 Shell 修改，也不会被写成 %%d
# ==============================================================================
cat << 'BODY_EOF' >> cmd/prompt.go

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
func buildContextPrompt() string {
	historyPath := "history/history.json"
	var items []promptHistoryItem
	
	// 1. 读取历史
	if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
		json.Unmarshal(data, &items)
	}

	var sb strings.Builder
	sb.WriteString("# Project Context (History)\n")
	
	// 2. 提取最新的项目结构
	var lastStructure string
	for _, item := range items {
		if val, ok := item.Context["project_structure"]; ok && val != "" {
			lastStructure = val
		}
	}

	// 3. 兜底逻辑 (带警告)
	if lastStructure == "" {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: Project structure missing from history. Using real-time file system snapshot instead.\n")
		if liveTree, err := fsutil.GenerateTree("."); err == nil {
			lastStructure = liveTree
		}
	}

	// 4. 生成历史摘要
	sb.WriteString("Recent development steps for context:\n\n")
	startIdx := 0
	if len(items) > 3 {
		startIdx = len(items) - 3
	}

	for i := startIdx; i < len(items); i++ {
		item := items[i]
		// 修复点：这里是 %d 和 %s，不是 %%d
		sb.WriteString(fmt.Sprintf("## History Step %d (%s)\n", i+1, item.Timestamp))
		
		shortPrompt := strings.ReplaceAll(item.OriginalPrompt, "\n", " ")
		if len(shortPrompt) > 80 {
			shortPrompt = shortPrompt[:80] + "..."
		}
		// 修复点：这里是 %s
		sb.WriteString(fmt.Sprintf("- Request: %s\n", shortPrompt)) 
		sb.WriteString(fmt.Sprintf("- Action: %s\n\n", item.Action))
	}
	
	// 5. 输出结构
	if lastStructure != "" {
		sb.WriteString("## Current Project Structure\n")
		sb.WriteString("```text\n")
		sb.WriteString(lastStructure)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("--------------------------------------------------\n")
	return sb.String()
}

// -----------------------------------------------------------
// 辅助函数: 附加输出格式约束
// -----------------------------------------------------------
func appendOutputConstraints(sb *strings.Builder, requirement string) {
	if outputFormat != "" && outputFormat != "text" {
		sb.WriteString(fmt.Sprintf("## Output Format Constraints\n"))
		// 修复点：这里是 %s
		sb.WriteString(fmt.Sprintf("1. Provide the solution as a **single %s file**.\n", outputFormat))
		sb.WriteString("2. **CRITICAL: METADATA HEADER REQUIRED**\n")
		sb.WriteString("   The file MUST start with a metadata header block in comments.\n")
		sb.WriteString("   The `original_prompt` field MUST contain the **EXACT FULL TEXT** of the 'New Requirement' section below. **DO NOT TRUNCATE.**\n")
		
		commentChar := "#"
		if outputFormat == "go" || outputFormat == "cpp" {
			commentChar = "//"
		}
		
		// 修复点：这里是 %s
		sb.WriteString(fmt.Sprintf("   Format: %s METADATA_START ... %s METADATA_END\n\n", commentChar, commentChar))
	}
}

// -----------------------------------------------------------
// 主命令: prompt
// -----------------------------------------------------------
var promptCmd = &cobra.Command{
	Use:   "prompt [requirement]",
	Short: "生成 AI 提示词 (任务模式)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var sb strings.Builder
		sb.WriteString(buildContextPrompt())
		
		sb.WriteString("# New Requirement (Current Task)\n")
		userRequirement := strings.Join(args, " ")
		sb.WriteString(userRequirement)
		sb.WriteString("\n\n")

		appendOutputConstraints(&sb, userRequirement)
		fmt.Println(sb.String())
	},
}

// -----------------------------------------------------------
// 子命令: prompt commit
// -----------------------------------------------------------
var promptCommitCmd = &cobra.Command{
	Use:   "commit [optional_instruction]",
	Short: "根据当前 Git 变更生成 Commit Message 提示词",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		diffCmd := exec.Command("git", "diff", "HEAD")
		diffOut, err := diffCmd.CombinedOutput()
		diffStr := string(diffOut)

		if err != nil {
			fmt.Printf("Error running git diff: %v\n", err)
			return
		}
		if len(strings.TrimSpace(diffStr)) == 0 {
			fmt.Println("❌ No changes detected (git diff is empty). Nothing to commit.")
			return
		}

		var sb strings.Builder
		
		sb.WriteString("# Task: Generate Git Commit Message\n")
		sb.WriteString("You are a Senior Developer. Please write a semantic git commit message for the following code changes.\n\n")
		
		sb.WriteString("## Code Changes (Git Diff)\n")
		sb.WriteString("```diff\n")
		if len(diffStr) > 8000 {
			sb.WriteString(diffStr[:8000] + "\n... (diff truncated) ...")
		} else {
			sb.WriteString(diffStr)
		}
		sb.WriteString("\n```\n\n")

		sb.WriteString(buildContextPrompt())

		sb.WriteString("## Instruction\n")
		instruction := "Analyze the diff above. Generate a concise and meaningful commit message following **Conventional Commits** format."
		if len(args) > 0 {
			instruction = strings.Join(args, " ")
		}
		sb.WriteString(instruction)
		sb.WriteString("\n\n")
		
		sb.WriteString("## Expected Output Format\n")
		sb.WriteString("```text\n")
		sb.WriteString("<type>(<scope>): <subject>\n")
		sb.WriteString("\n")
		sb.WriteString("<body>\n")
		sb.WriteString("\n")
		sb.WriteString("[Optional Footer: Ref #IssueID]\n")
		sb.WriteString("```\n")

		fmt.Println(sb.String())
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.AddCommand(promptCommitCmd)
	promptCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "shell", "Expected output format")
}
BODY_EOF

echo -e "${GREEN}-> 重新编译...${NC}"
make build

echo -e "${GREEN}=== 修复完成 ===${NC}"
echo -e "现在运行 ${GREEN}bin/cli prompt \"check\"${NC}，格式应该完美了。"
