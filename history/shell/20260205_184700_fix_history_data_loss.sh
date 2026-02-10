#!/bin/bash
# ---------------------------------------------------------
# METADATA_START
# Timestamp: 2026-02-05 18:47:00
# OriginalPrompt: 你又忘记了shell的meta字段 还有个问题为什么我需要给这个shell的操作像其他的shell一样取个名字然后注册请你修复就像其他shell一样
# Summary: 紧急修复 history.json 数据丢失漏洞
# Action: 1. 修正 internal/meta/parser.go 增加 FileChanges 字段防止序列化丢失 2. 修正 cmd/history_add.go 增加 JSON 解析错误熔断机制 3. 重新编译并注册本脚本
# ExpectedOutcome: 执行 make register 或 history add 时，不再因字段缺失或解析错误而清空现有历史记录
# METADATA_END
# ---------------------------------------------------------

set -e

# 1. 修复 internal/meta/parser.go
# 增加 FileChanges 结构体定义，并在 HistoryItem 中引用，确保该字段不会被丢弃
printf "➜  Patching internal/meta/parser.go...\n"
cat > internal/meta/parser.go << 'GO_CODE'
package meta

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// FileChanges 记录文件变更详情
type FileChanges struct {
	Created   []string `json:"created,omitempty"`
	Modified  []string `json:"modified,omitempty"`
	Completed []string `json:"completed,omitempty"`
	Pending   []string `json:"pending,omitempty"`
}

// HistoryItem 对应 history.json 的结构
type HistoryItem struct {
	Timestamp       string            `json:"timestamp"`
	OriginalPrompt  string            `json:"original_prompt"`
	Summary         string            `json:"summary"`
	Action          string            `json:"action"`
	ExpectedOutcome string            `json:"expected_outcome"`
	// 新增 Context 字段，用于存储 project_structure 等额外信息
	Context         map[string]string `json:"context,omitempty"`
	// 新增 FileChanges 字段，防止回写时丢失
	FileChanges     *FileChanges      `json:"file_changes,omitempty"`
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
GO_CODE

# 2. 修复 cmd/history_add.go
# 增加了 json.Unmarshal 的错误处理，防止因解析失败导致的空切片覆盖文件
printf "➜  Patching cmd/history_add.go...\n"
cat > cmd/history_add.go << 'GO_CODE'
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"github.com/A-Flex-Box/cli/internal/meta"
	"github.com/A-Flex-Box/cli/internal/fsutil"

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
				// ★★★ 核心修复：检查 Unmarshal 错误 ★★★
				if err := json.Unmarshal(data, &items); err != nil {
					fmt.Printf("❌ CRITICAL ERROR: Failed to parse existing history.json: %v\n", err)
					fmt.Printf("🛑 Aborting operation to prevent data loss. Please fix the JSON file manually.\n")
					os.Exit(1)
				}
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
GO_CODE

# 3. 重新编译项目
printf "➜  Rebuilding CLI to apply fixes...\n"
make build

# 4. 自我注册 (此时 bin/cli 已是修复后的版本，注册是安全的)
printf "➜  Registering fix script to history...\n"
./bin/cli history add "$0"

printf "✅ Fix applied and registered successfully.\n"
