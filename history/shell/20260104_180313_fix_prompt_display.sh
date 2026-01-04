#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 19:50:00
# original_prompt: 存在是你没显示而已
# summary: 修复 prompt 命令中项目结构无法显示的问题
# action: 重写 cmd/prompt.go，使用 Quoted Heredoc 策略防止 Shell 吞掉 Go 代码中的反引号，确保 JSON Struct Tag 和 Markdown 代码块逻辑正确
# expected_outcome: 运行 bin/cli prompt 将能正确显示 history.json 中已存在的 project_structure 树形图
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}-> 正在修复 cmd/prompt.go (显示逻辑)...${NC}"

# 获取模块名
MODULE_NAME=$(grep "^module" go.mod | awk '{print $2}')

# 第一步：写入头部（需要变量扩展，所以用 EOF）
cat << EOF > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"[github.com/spf13/cobra](https://github.com/spf13/cobra)"
)
EOF

# 第二步：写入主体（使用 'EOF' 彻底禁用 Shell 扩展，保护反引号）
cat << 'EOF' >> cmd/prompt.go

// promptHistoryItem 必须匹配 history.json 的结构
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
		
		// 容错读取
		if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
			json.Unmarshal(data, &items)
		}

		var sb strings.Builder
		sb.WriteString("# Context: Project Development History\n")
		sb.WriteString("I am working on a Go CLI tool. Here is the summary of previous steps:\n\n")

		var lastStructure string

		for i, item := range items {
			sb.WriteString(fmt.Sprintf("## Step %d (%s)\n", i+1, item.Timestamp))
			
			// 简单的清理换行，避免破坏列表格式
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

		// 如果找到了结构快照，打印出来
		if lastStructure != "" {
			sb.WriteString("## Current Project Structure\n")
			// 这里使用了反引号，上一版脚本可能在这里出错了
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

		// Output Format Constraints
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

echo -e "${GREEN}-> 重新编译...${NC}"
go mod tidy
make build

echo -e "${GREEN}=== 修复完成 ===${NC}"
echo -e "现在请再次运行: ${GREEN}bin/cli prompt \"查看结构\" -f shell${NC}"
echo -e "你应该能看到 '## Current Project Structure' 下方出现了完整的文件树。"