package history

import (
	"fmt"
	"os"
	"strings"

	"github.com/A-Flex-Box/cli/pkgs"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "add [file]",
		Short:   "提取文件元数据，生成目录快照并存入 history.json",
		Example: "cli history add ./script.sh",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			if err := Add(pkgs.DefaultHistoryPath, filePath); err != nil {
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
}
