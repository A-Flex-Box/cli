#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 20:50:00
# original_prompt: 你上面那个修复builder的panic你还没修复呢把那个脚本用新的方式输出
# summary: 修复 cmd/prompt.go 中 strings.Builder 值拷贝导致的 Panic (One-Click 模式)
# action: 重写 cmd/prompt.go，将 buildContextPrompt 函数返回值改为 string，并在主逻辑中通过 WriteString 拼接，避免非法拷贝。
# expected_outcome: bin/cli prompt commit 命令能够正常执行，不再报错。
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在修复 cmd/prompt.go (解决 strings.Builder Copy Panic)...${NC}"

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
// 修改点：返回 string 而不是 strings.Builder，避免 Copy Panic
// -----------------------------------------------------------
func buildContextPrompt() string {
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
		
		// 简单的清理换行
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
	return sb.String() // 返回 string，防止 copy
}

// 辅助函数: 附加输出格式约束 (接收指针，安全)
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
		// 创建一个新的 Builder
		var sb strings.Builder
		// 写入 Context 字符串
		sb.WriteString(buildContextPrompt())
		
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
	Long:  `获取当前工作区所有未提交的变更 (git diff HEAD)，并将其加入到 Prompt 上下文中。`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var sb strings.Builder
		sb.WriteString(buildContextPrompt())

		// 获取 Git Diff
		diffCmd := exec.Command("git", "diff", "HEAD")
		diffOut, err := diffCmd.CombinedOutput()
		
		sb.WriteString("# Current Code Changes (Git Diff)\n")
		sb.WriteString("I have made the following changes to the codebase:\n\n")
		sb.WriteString("```diff\n")
		
		if err != nil {
			sb.WriteString(fmt.Sprintf("Error reading diff: %v\n", err))
		} else {
			diffStr := string(diffOut)
			// 简单的截断保护，防止 prompt 过长
			if len(diffStr) > 6000 {
				sb.WriteString(diffStr[:6000] + "\n... (diff truncated) ...")
			} else {
				sb.WriteString(diffStr)
			}
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
	promptCmd.AddCommand(promptCommitCmd)
	promptCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "shell", "Expected output format")
}
GO_EOF

echo -e "${GREEN}-> 重新编译...${NC}"
make build

echo -e "${GREEN}=== 修复完成，Panic 已解决 ===${NC}"
