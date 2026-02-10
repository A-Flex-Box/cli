package cmd

import (
	"github.com/A-Flex-Box/cli/app/ai"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI 工程化辅助工具",
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "检查 GPU、CUDA 及虚拟环境列表",
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		defer log.Sync()
		ai.Setup(log)
	},
}

var initCmd = &cobra.Command{
	Use:   "init [project_name]",
	Short: "生成 AI 项目标准目录结构",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		defer log.Sync()
		ai.Init(log, args[0])
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(setupCmd)
	aiCmd.AddCommand(initCmd)
}
