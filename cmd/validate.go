package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/A-Flex-Box/cli/internal/meta"

	"github.com/spf13/cobra"
)

var (
	isAnswer bool
	lang     string
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "校验文件格式或 AI 回答规范",
	Long:  `校验指定文件。如果指定 --answer，将严格检查是否包含符合项目规范的历史元数据头。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		
		// 1. 基础文件存在性校验
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("❌ Error: File '%s' does not exist.\n", filePath)
			os.Exit(1)
		}

		fmt.Printf("🔍 Validating '%s'...\n", filePath)

		// 2. 如果是 Answer 模式，校验元数据
		if isAnswer {
			// 自动推断语言 (如果未指定)
			if lang == "" {
				ext := filepath.Ext(filePath)
				if len(ext) > 1 {
					lang = ext[1:] // remove dot
				} else {
					lang = "shell" // default
				}
			}

			item, err := meta.ParseMetadata(filePath, lang)
			if err != nil {
				fmt.Printf("❌ Metadata Validation Failed:\n   %v\n", err)
				fmt.Println("   Ensure the file contains a header like:")
				fmt.Println("   # METADATA_START")
				fmt.Println("   # timestamp: ...")
				fmt.Println("   # ...")
				fmt.Println("   # METADATA_END")
				os.Exit(1)
			}

			fmt.Println("✅ Metadata Header is Valid:")
			fmt.Printf("   - Timestamp: %s\n", item.Timestamp)
			fmt.Printf("   - Summary:   %s\n", item.Summary)
		}

		// 3. 这里可以预留接口做具体语言的 Syntax Check
		// 比如调用 go fmt 或 shfmt (如有)
		
		fmt.Println("✅ File validation passed.")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().BoolVar(&isAnswer, "answer", false, "Validate as an AI answer (require metadata)")
	validateCmd.Flags().StringVarP(&lang, "format", "f", "", "Source language (shell, go, python, etc.)")
}
