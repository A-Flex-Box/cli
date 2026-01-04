#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 21:40:00
# original_prompt: ➜ cli git:(master) ✗ >.... (heredoc error)
# summary: 修复粘贴截断导致的 Heredoc 错误，并重新应用 Prompt 模板修复
# action: 重新生成 cmd/prompt.go，移除误导性的 footer 标签，确保 Go 代码完整闭合，并重新编译。
# expected_outcome: 脚本正常执行，不再卡在 heredoc，bin/cli prompt commit 生成标准格式。
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在重写 cmd/prompt.go (修复 Footer 标签并解决粘贴错误)...${NC}"

# 使用 GO_EOF 作为 Go 代码块的定界符
cat << 'GO_EOF' > cmd/prompt.go
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
func buildContextPrompt() string {
	historyPath := "history/history.json"
	var items []promptHistoryItem
	
	if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
		json.Unmarshal(data, &items)
	}

	var sb strings.Builder
	sb.WriteString("# Project Context (History)\n")
	sb.WriteString("Recent development steps for context:\n\n")

	startIdx := 0
	if len(items) > 3 {
		startIdx = len(items) - 3
	}

	for i := startIdx; i < len(items); i++ {
		item := items[i]
		sb.WriteString(fmt.Sprintf("## History Step %d (%s)\n", i+1, item.Timestamp))
		shortPrompt := strings.ReplaceAll(item.OriginalPrompt, "\n", " ")
		if len(shortPrompt) > 80 {
			shortPrompt = shortPrompt[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("- Request: %s\n", shortPrompt)) 
		sb.WriteString(fmt.Sprintf("- Action: %s\n\n", item.Action))
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
		sb.WriteString(fmt.Sprintf("1. Provide the solution as a **single %s file**.\n", outputFormat))
		sb.WriteString("2. **CRITICAL: METADATA HEADER REQUIRED**\n")
		sb.WriteString("   The file MUST start with a metadata header block in comments.\n")
		sb.WriteString("   The `original_prompt` field MUST contain the **EXACT FULL TEXT** of the 'New Requirement' section below. **DO NOT TRUNCATE.**\n")
		
		commentChar := "#"
		if outputFormat == "go" || outputFormat == "cpp" {
			commentChar = "//"
		}
		
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
		
		// 修正点：移除了 literal footer 标签
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
GO_EOF

echo -e "${GREEN}-> 重新编译...${NC}"
make build

echo -e "${GREEN}=== 修复完成 ===${NC}"
echo -e "请运行: ${GREEN}bin/cli prompt commit${NC}"
