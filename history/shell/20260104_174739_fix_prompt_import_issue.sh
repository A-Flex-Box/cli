#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 19:00:00
# original_prompt: ➜  cli git:(master) ✗ chmod +x fix_prompt.sh 
# ➜  cli git:(master) ✗ ./fix_prompt.sh 
# -> 正在修补 cmd/prompt.go 以强制保留原始提问...
# ➜  Compiling...
# cmd/prompt.go:9:2: no required module provides package github.com/spf13/viper; to add it:
#         go get github.com/spf13/viper
# make: *** [Makefile:31: build] Error 1
# ➜  cli git:(master) ✗ 
# summary: 修复 cmd/prompt.go 中未使用的 viper 包导入导致的编译错误
# action: 重写 cmd/prompt.go 移除 github.com/spf13/viper，执行 go mod tidy 清理依赖，重新执行 make build
# expected_outcome: 项目成功编译，bin/cli 可用
# METADATA_END

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在修复 cmd/prompt.go (移除未使用的 import)...${NC}"

# 重写文件，移除 viper
cat << 'EOF' > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Simple structure to read history
type simpleHistoryItem struct {
	Timestamp       string `json:"timestamp"`
	OriginalPrompt  string `json:"original_prompt"`
	Summary         string `json:"summary"`
	Action          string `json:"action"`
	ExpectedOutcome string `json:"expected_outcome"`
}

var outputFormat string

var promptCmd = &cobra.Command{
	Use:   "prompt [requirement]",
	Short: "生成包含项目上下文的 AI 提示词",
	Long:  `读取 history.json，结合当前需求，生成 Prompt。`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 读取 History
		historyPath := "history/history.json"
		var items []simpleHistoryItem
		
		if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
			// 忽略错误，尽可能解析
			json.Unmarshal(data, &items)
		}

		// 2. 构建 Prompt Context
		var sb strings.Builder
		sb.WriteString("# Context: Project Development History\n")
		sb.WriteString("I am working on a Go CLI tool. Here is the summary of previous steps:\n\n")

		for i, item := range items {
			sb.WriteString(fmt.Sprintf("## Step %d (%s)\n", i+1, item.Timestamp))
			
			// 历史记录摘要显示（仅用于 Context 阅读，不影响下面生成的要求）
			shortPrompt := item.OriginalPrompt
			if len(shortPrompt) > 100 {
				shortPrompt = shortPrompt[:100] + "..."
			}
			// 处理换行符，避免破坏 Markdown 列表格式
			shortPrompt = strings.ReplaceAll(shortPrompt, "\n", " ")

			sb.WriteString(fmt.Sprintf("- **Prompt Summary**: %s\n", shortPrompt)) 
			sb.WriteString(fmt.Sprintf("- **Action**: %s\n", item.Action))
			sb.WriteString(fmt.Sprintf("- **Outcome**: %s\n\n", item.ExpectedOutcome))
		}

		sb.WriteString("--------------------------------------------------\n")
		sb.WriteString("# New Requirement (Current Task)\n")
		sb.WriteString("Based on the context, please fulfill the following:\n\n")
		
		userRequirement := strings.Join(args, " ")
		sb.WriteString(userRequirement)
		sb.WriteString("\n\n")

		// 3. 添加格式与元数据约束
		if outputFormat != "" {
			sb.WriteString(fmt.Sprintf("## Output Format Constraints\n"))
			sb.WriteString(fmt.Sprintf("1. Provide the solution as a **single %s file**.\n", outputFormat))
			sb.WriteString("2. **CRITICAL: METADATA HEADER REQUIRED**\n")
			sb.WriteString("   The file MUST start with a metadata header block in comments.\n")
			sb.WriteString("   The `original_prompt` field MUST contain the **EXACT FULL TEXT** of the 'New Requirement' section above. **DO NOT TRUNCATE, DO NOT SUMMARIZE.**\n\n")
			
			// 根据格式提示注释符号
			commentChar := "#"
			if outputFormat == "go" || outputFormat == "cpp" {
				commentChar = "//"
			}
			
			sb.WriteString("   Template:\n")
			sb.WriteString(fmt.Sprintf("   %s METADATA_START\n", commentChar))
			sb.WriteString(fmt.Sprintf("   %s timestamp: <YYYY-MM-DD HH:MM:SS>\n", commentChar))
			sb.WriteString(fmt.Sprintf("   %s original_prompt: %s\n", commentChar, userRequirement)) 
			sb.WriteString(fmt.Sprintf("   %s summary: <Short summary of what you did>\n", commentChar))
			sb.WriteString(fmt.Sprintf("   %s action: <Technical actions taken>\n", commentChar))
			sb.WriteString(fmt.Sprintf("   %s expected_outcome: <What this script achieves>\n", commentChar))
			sb.WriteString(fmt.Sprintf("   %s METADATA_END\n\n", commentChar))
		}

		fmt.Println(sb.String())
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.Flags().StringVarP(&outputFormat, "format", "f", "shell", "Expected output format")
}
EOF

echo -e "${GREEN}-> 清理依赖 (go mod tidy)...${NC}"
go mod tidy

echo -e "${GREEN}-> 重新编译 (make build)...${NC}"
make build

echo -e "${GREEN}=== 修复完成，编译成功！ ===${NC}"