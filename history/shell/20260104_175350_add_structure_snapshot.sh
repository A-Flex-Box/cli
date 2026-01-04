#!/bin/bash
# METADATA_START
# timestamp: 2026-01-04 19:30:00
# original_prompt: 我还需要在元数据加一个可选map字段里面首先需要加一个项目结构这个用来展示对应的这次操作完成后的项目结构变成什么了,register里面也应该加入这个功能也就是将当前的项目文件层级关系记录下来
# summary: 架构升级：在历史记录中自动捕获并存储项目文件结构快照
# action: 新增 fsutil 包实现 tree 功能，更新 meta 结构体支持 Context 字段，修改 history add 命令自动注入 project_structure，更新 prompt 命令展示最新结构
# expected_outcome: 每次 make register 后，history.json 会包含当时的目录结构快照，且下一次 prompt 生成时能看到该结构
# METADATA_END

set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== 正在升级 CLI: 添加项目结构快照功能 ===${NC}"

# 获取模块名
MODULE_NAME=$(grep "^module" go.mod | awk '{print $2}')

# ==========================================
# 1. 创建 internal/fsutil (Tree 生成器)
# ==========================================
echo -e "${GREEN}-> [1/5] 创建 internal/fsutil 包...${NC}"
mkdir -p internal/fsutil

cat << EOF > internal/fsutil/tree.go
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateTree 生成项目目录结构的字符串表示
// 忽略 .git, bin, history/shell (为了保持 JSON 简洁)
func GenerateTree(root string) (string, error) {
	var sb strings.Builder
	sb.WriteString(".\n")

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if path == root {
			return nil
		}

		// 过滤规则
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "bin" {
				return filepath.SkipDir
			}
			// history 目录我们要，但是 history/shell 里面的脚本太多了，可以忽略内容只看目录
			if path == "history/shell" {
				// 记录目录本身，但跳过子内容
				indent := strings.Repeat("│   ", strings.Count(path, string(os.PathSeparator)))
				sb.WriteString(fmt.Sprintf("%s├── %s/ (archived scripts hidden)\n", indent, info.Name()))
				return filepath.SkipDir
			}
		}

		// 计算缩进
		relPath, _ := filepath.Rel(root, path)
		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("│   ", depth)
		
		marker := "├── "
		// 这里简化处理，不完美区分最后一个节点（└──），为了代码短小
		
		displayName := info.Name()
		if info.IsDir() {
			displayName += "/"
		}

		sb.WriteString(fmt.Sprintf("%s%s%s\n", indent, marker, displayName))
		return nil
	})

	return sb.String(), err
}
EOF

# ==========================================
# 2. 更新 internal/meta/parser.go (结构体变更)
# ==========================================
echo -e "${GREEN}-> [2/5] 更新 HistoryItem 结构体...${NC}"

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
	Timestamp       string            \`json:"timestamp"\`
	OriginalPrompt  string            \`json:"original_prompt"\`
	Summary         string            \`json:"summary"\`
	Action          string            \`json:"action"\`
	ExpectedOutcome string            \`json:"expected_outcome"\`
	// 新增 Context 字段，用于存储 project_structure 等额外信息
	Context         map[string]string \`json:"context,omitempty"\`
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
	"go":     {"//"},
	"cpp":    {"//"},
}

// ParseMetadata 读取文件并提取 Metadata Block
func ParseMetadata(filePath string, lang string) (*HistoryItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	config, ok := langConfigs[strings.ToLower(lang)]
	if !ok {
		config = LanguageConfig{CommentPrefix: "#"}
	}
	prefix := config.CommentPrefix

	scanner := bufio.NewScanner(file)
	inBlock := false
	item := &HistoryItem{
		Context: make(map[string]string),
	}
	foundFields := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.Contains(line, "METADATA_START") {
			inBlock = true
			continue
		}
		if strings.Contains(line, "METADATA_END") {
			break
		}

		if inBlock {
			content := strings.TrimPrefix(line, prefix)
			content = strings.TrimSpace(content)

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
				case "expected_outcome":
					item.ExpectedOutcome = val
					foundFields++
				// 其他字段可以放入 Context，暂时只解析核心字段
				}
			}
		}
	}

	if foundFields < 3 {
		return nil, fmt.Errorf("metadata incomplete (found %d fields)", foundFields)
	}

	return item, nil
}
EOF

# ==========================================
# 3. 更新 cmd/history_add.go (自动注入结构快照)
# ==========================================
echo -e "${GREEN}-> [3/5] 更新 history add 逻辑 (自动注入 Project Structure)...${NC}"

cat << EOF > cmd/history_add.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"${MODULE_NAME}/internal/meta"
	"${MODULE_NAME}/internal/fsutil"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage project history",
}

var historyAddCmd = &cobra.Command{
	Use:   "add [file]",
	Short: "提取文件元数据，生成目录快照并存入 history.json",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		
		// 1. 提取元数据
		lang := "shell"
		if ext := filepath.Ext(filePath); len(ext) > 1 {
			lang = ext[1:]
		}
		
		newItem, err := meta.ParseMetadata(filePath, lang)
		if err != nil {
			fmt.Printf("❌ Failed to extract metadata: %v\n", err)
			os.Exit(1)
		}

		// 2. ★★★ 生成项目结构快照 ★★★
		treeStr, err := fsutil.GenerateTree(".")
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to generate project structure: %v\n", err)
		} else {
			if newItem.Context == nil {
				newItem.Context = make(map[string]string)
			}
			newItem.Context["project_structure"] = treeStr
			fmt.Println("📸 Project structure snapshot captured.")
		}

		// 3. 读取现有 History
		historyPath := "history/history.json"
		var items []meta.HistoryItem

		if _, err := os.Stat(historyPath); err == nil {
			data, err := os.ReadFile(historyPath)
			if err == nil && len(data) > 0 {
				json.Unmarshal(data, &items)
			}
		}

		// 4. 追加
		items = append(items, *newItem)

		// 5. 回写
		newData, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			fmt.Printf("❌ Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(historyPath, newData, 0644); err != nil {
			fmt.Printf("❌ Error writing history file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ History updated from '%s'.\n", filePath)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.AddCommand(historyAddCmd)
}
EOF

# ==========================================
# 4. 更新 cmd/prompt.go (展示最新结构)
# ==========================================
echo -e "${GREEN}-> [4/5] 更新 prompt 逻辑 (展示 Context 中的结构)...${NC}"

cat << EOF > cmd/prompt.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// 必须与 meta 包结构一致
type promptHistoryItem struct {
	Timestamp       string            \`json:"timestamp"\`
	OriginalPrompt  string            \`json:"original_prompt"\`
	Summary         string            \`json:"summary"\`
	Action          string            \`json:"action"\`
	ExpectedOutcome string            \`json:"expected_outcome"\`
	Context         map[string]string \`json:"context,omitempty"\`
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
			if len(shortPrompt) > 120 {
				shortPrompt = shortPrompt[:120] + "..."
			}

			sb.WriteString(fmt.Sprintf("- **Prompt Summary**: %s\n", shortPrompt)) 
			sb.WriteString(fmt.Sprintf("- **Action**: %s\n", item.Action))
			sb.WriteString(fmt.Sprintf("- **Outcome**: %s\n\n", item.ExpectedOutcome))

			// 记录最新的结构快照
			if val, ok := item.Context["project_structure"]; ok && val != "" {
				lastStructure = val
			}
		}

		// 如果有结构快照，展示在 Context 最后
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

		// Output Constraints
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

# ==========================================
# 5. 编译与清理
# ==========================================
echo -e "${GREEN}-> [5/5] 重新编译项目...${NC}"
go mod tidy
make build

echo -e "\n${GREEN}=== 升级完成！ ===${NC}"
echo -e "请务必运行以下命令来'自我注册'这个脚本，以便首次生成目录快照："
echo -e "${BLUE}make register FILE=add_structure_snapshot.sh${BLUE}"