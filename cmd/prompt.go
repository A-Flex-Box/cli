package cmd

import (
	"fmt"
	"strings"

	"github.com/A-Flex-Box/cli/app/prompt"
	"github.com/A-Flex-Box/cli/pkgs"
	"github.com/spf13/cobra"
)

var outputFormat string

var promptCmd = &cobra.Command{
	Use:   "prompt [requirement]",
	Short: "生成 AI 提示词 (任务模式)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s := prompt.GenerateTaskPrompt(pkgs.DefaultHistoryPath, strings.Join(args, " "), outputFormat)
		fmt.Println(s)
	},
}

var promptCommitCmd = &cobra.Command{
	Use:   "commit [optional_instruction]",
	Short: "根据当前 Git 变更生成 Commit Message 提示词",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		instruction := ""
		if len(args) > 0 {
			instruction = strings.Join(args, " ")
		}
		s, err := prompt.GenerateCommitPrompt(pkgs.DefaultHistoryPath, instruction)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			if strings.Contains(err.Error(), "no changes") {
				fmt.Println("❌ No changes detected (git diff is empty). Nothing to commit.")
			}
			return
		}
		fmt.Println(s)
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
	promptCmd.AddCommand(promptCommitCmd)
	promptCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "shell", "Expected output format")
}
