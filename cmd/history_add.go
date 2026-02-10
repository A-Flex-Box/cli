package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/A-Flex-Box/cli/app/history"
	"github.com/A-Flex-Box/cli/pkgs"
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
		if err := history.Add(pkgs.DefaultHistoryPath, filePath); err != nil {
			fmt.Printf("❌ %v\n", err)
			if strings.Contains(err.Error(), "parse") {
				fmt.Println("🛑 Aborting operation to prevent data loss. Please fix history.json manually.")
			}
			os.Exit(1)
		}
		fmt.Println("📸 Project structure snapshot captured.")
		fmt.Printf("✅ History updated from '%s'.\n", filePath)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.AddCommand(historyAddCmd)
}
