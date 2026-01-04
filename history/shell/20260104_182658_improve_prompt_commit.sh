#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 21:15:00
# original_prompt: 对于commit来说应该明确指出要生成的是commit commit的生成是不需要任务参数的就是现在未提交的内容变更,以及可选的原先操作的历史
# summary: 优化 prompt commit 子命令体验 (Diff-Driven Development)
# action: 修改 cmd/prompt.go，使 commit 子命令参数可选（默认为生成提交信息），增加空 Diff 检测，优化 Prompt 模板结构以 Diff 为核心。
# expected_outcome: 运行 bin/cli prompt commit 即可直接生成请求写 Commit Message 的提示词，无需额外参数。
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在重构 cmd/prompt.go (优化 commit 逻辑)...${NC}"

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
// 辅助函数: 生成基础 Context Prompt (返回 string)
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

	// 只取最近的 3 条历史，避免 Prompt 过长干扰 Commit 生成，除非需要完整上下文
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
// 优化点：Diff 优先，参数可选，默认生成 Commit Message
// -----------------------------------------------------------
var promptCommitCmd = &cobra.Command{
	Use:   "commit [optional_instruction]",
	Short: "根据当前 Git 变更生成 Commit Message 提示词",
	Long:  `获取 git diff，生成请求 AI 编写 Commit Message 的 Prompt。如果不传参数，默认指令为生成标准提交信息。`,
	Args:  cobra.ArbitraryArgs, // 允许 0 个或多个参数
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 获取 Diff
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
		
		// 2. 构建 Prompt - 结构调整：Diff 在最前，History 在后辅助
		sb.WriteString("# Task: Generate Git Commit Message\n")
		sb.WriteString("You are a Senior Developer. Please write a semantic git commit message for the following code changes.\n\n")
		
		// 3. 插入 Diff
		sb.WriteString("## Code Changes (Git Diff)\n")
		sb.WriteString("```diff\n")
		if len(diffStr) > 8000 {
			sb.WriteString(diffStr[:8000] + "\n... (diff truncated) ...")
		} else {
			sb.WriteString(diffStr)
		}
		sb.WriteString("\n```\n\n")

		// 4. 插入 Context (可选，辅助理解)
		sb.WriteString(buildContextPrompt())

		// 5. 插入具体指令
		sb.WriteString("## Instruction\n")
		instruction := "Analyze the diff above. Generate a concise and meaningful commit message following **Conventional Commits** format (feat, fix, docs, style, refactor, test, chore)."
		
		// 如果用户给了参数，覆盖默认指令
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
		sb.WriteString("<footer>\n")
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

echo -e "${GREEN}=== 升级完成 ===${NC}"
echo -e "现在的用法："
echo -e "1. ${GREEN}bin/cli prompt commit${NC} (默认: 帮我写 commit message)"
echo -e "2. ${GREEN}bin/cli prompt commit \"重点关注 Makefile 的修改\"${NC} (自定义侧重)"
