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
