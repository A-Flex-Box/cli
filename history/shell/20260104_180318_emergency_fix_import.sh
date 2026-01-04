#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 20:05:00
# original_prompt: ➜  cli git:(master) ✗ ./fix_prompt_display.sh    
# -> 正在修复 cmd/prompt.go (显示逻辑)...
# -> 重新编译...
# go: github.com/A-Flex-Box/cli/cmd imports
#         [github.com/spf13/cobra](https://github.com/spf13/cobra): malformed import path "[github.com/spf13/cobra](https://github.com/spf13/cobra)": double slash
# ➜  cli git:(master) ✗ 
# summary: 紧急修复 cmd/prompt.go 中被意外格式化为 Markdown 链接的 import 路径
# action: 强制重写 cmd/prompt.go，确保 import 部分为纯文本，移除任何 Markdown 语法残留，重新编译
# expected_outcome: 消除 "malformed import path" 错误，成功编译并保留项目结构显示功能
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在清洗 cmd/prompt.go (移除 Markdown 链接污染)...${NC}"

# ========================================================
# 第一步：写入干净的 Import 块
# 使用 EOF (不带引号) 允许变量扩展（虽然这里没用到变量，但习惯上保持一致）
# 关键：这里的 import 路径是纯文本，绝对不要带中括号或 http 链接
# ========================================================
cat << EOF > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)
EOF

# ========================================================
# 第二步：追加业务逻辑
# 使用 'EOF' (带单引号) 彻底禁用 Shell 扩展
# 这样 Go 代码里的反引号 (Struct Tag) 才能被原样写入
# ========================================================
cat << 'EOF' >> cmd/prompt.go

// promptHistoryItem 对应 history.json 的结构
type promptHistoryItem struct {
	Timestamp       string            `json:"timestamp"`
	OriginalPrompt  string            `json:"original_prompt"`
	Summary         string            `json:"summary"`
	Action          string            `json:"action"`
	ExpectedOutcome string            `json:"expected_outcome"`
	Context         map[string]string `json:"context,omitempty"`
}

var outputFormat string

var promptCmd = &cobra.Command{
	Use:   "prompt [requirement]",
	Short: "生成包含项目上下文的 AI 提示词",
	Run: func(cmd *cobra.Command, args []string) {
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

			// 捕获最新的结构快照
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
		sb.WriteString("# New Requirement (Current Task)\n")
		sb.WriteString("Based on the context, please fulfill the following:\n\n")
		
		userRequirement := strings.Join(args, " ")
		sb.WriteString(userRequirement)
		sb.WriteString("\n\n")

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
			sb.WriteString(fmt.Sprintf("   %s original_prompt: %s\n", commentChar, userRequirement)) 
			sb.WriteString(fmt.Sprintf("   %s summary: <Short summary>\n", commentChar))
			sb.WriteString(fmt.Sprintf("   %s action: <Actions taken>\n", commentChar))
			sb.WriteString(fmt.Sprintf("   %s expected_outcome: <Outcome>\n", commentChar))
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

echo -e "${GREEN}-> 清理依赖并重新编译...${NC}"
go mod tidy
make build

echo -e "${GREEN}=== 修复完成 ===${NC}"
echo -e "现在可以运行: ${GREEN}bin/cli prompt \"查看结构\" -f shell${NC}"