package validate

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCmd returns the validate command.
func NewCmd() *cobra.Command {
	var isAnswer bool
	var lang string

	cmd := &cobra.Command{
		Use:     "validate [file]",
		Short:   "校验文件格式或 AI 回答规范",
		Long:    `校验指定文件。如果指定 --answer，将严格检查是否包含符合项目规范的历史元数据头。`,
		Example: "cli validate answer.go --answer",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			res := ValidateFile(filePath, isAnswer, lang)

			if !res.Exists {
				fmt.Printf("❌ Error: %v\n", res.Err)
				os.Exit(1)
			}

			fmt.Printf("🔍 Validating '%s'...\n", filePath)

			if isAnswer && !res.OK {
				fmt.Printf("❌ Metadata Validation Failed:\n   %v\n", res.Err)
				fmt.Println("   Ensure the file contains a header like:")
				fmt.Println("   # METADATA_START")
				fmt.Println("   # timestamp: ...")
				fmt.Println("   # ...")
				fmt.Println("   # METADATA_END")
				os.Exit(1)
			}

			if isAnswer && res.Item != nil {
				fmt.Println("✅ Metadata Header is Valid:")
				fmt.Printf("   - Timestamp: %s\n", res.Item.Timestamp)
				fmt.Printf("   - Summary:   %s\n", res.Item.Summary)
				if res.Item.Iteration != "" {
					fmt.Printf("   - Iteration: %s\n", res.Item.Iteration)
				}
			}

			fmt.Println("✅ File validation passed.")
		},
	}
	cmd.Flags().BoolVar(&isAnswer, "answer", false, "Validate as an AI answer (require metadata)")
	cmd.Flags().StringVarP(&lang, "format", "f", "", "Source language (shell, go, python, etc.)")
	return cmd
}
