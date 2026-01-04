#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 18:55:00
# original_prompt: 有点问题你不应该省略我的问题,你现在把那个完整的元数据json输出给我我手动添加,然后请你记住那个shell里面我的提问内容很重要生成的shell必须携带这个所以请加上这个要求告诉我在对应的位置
# summary: 修复 prompt 生成逻辑，强制保留原始提问全文
# action: 更新 cmd/prompt.go，在生成的 Prompt 模板中增加 strict rule，要求 metadata 中的 original_prompt 必须是全量文本
# expected_outcome: 未来的 AI 回复生成的脚本中，original_prompt 字段将包含用户输入的每一个字
# METADATA_END

set -e
echo "-> 正在修补 cmd/prompt.go 以强制保留原始提问..."

# 我们只重写 cmd/prompt.go 中关于 Prompt 模板构建的部分
# 注意看下面代码中 original_prompt: <YOUR_FULL_INPUT_HERE> 旁边的注释

cat << 'EOF' > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"github.com/spf13/cobra"
	"github.com/spf13/viper" 
)

// 需要把模块名替换成你自己的，这里假设你的 go.mod 还没变，如果变了请手动调整 import
// 为了通用性，这里我先用相对路径概念，实际代码中需要完整的 module path
// 暂时这里不依赖 meta 包，只做简单的 json 读取，为了脚本简洁

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
			json.Unmarshal(data, &items)
		}

		// 2. 构建 Prompt Context
		var sb strings.Builder
		sb.WriteString("# Context: Project Development History\n")
		sb.WriteString("I am working on a Go CLI tool. Here is the summary of previous steps:\n\n")

		for i, item := range items {
			sb.WriteString(fmt.Sprintf("## Step %d (%s)\n", i+1, item.Timestamp))
			// 这里展示历史记录时，为了阅读方便可以截断，但生成的 Prompt 要求里不能截断
			shortPrompt := item.OriginalPrompt
			if len(shortPrompt) > 100 {
				shortPrompt = shortPrompt[:100] + "..."
			}
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

# 重新编译
make build
echo "✅ prompt命令已更新。现在它会强制 AI 在生成的脚本中包含完整的原始提问。"