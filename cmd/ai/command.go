package ai

import (
	"github.com/A-Flex-Box/cli/app/ai"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/cobra"
)

// NewCmd returns the ai command with setup and init subcommands.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI 工程化辅助工具",
	}
	cmd.AddCommand(newSetupCmd(), newInitCmd())
	return cmd
}

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "检查 GPU、CUDA 及虚拟环境列表",
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.NewLogger()
			defer log.Sync()
			ai.Setup(log)
		},
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [project_name]",
		Short: "生成 AI 项目标准目录结构",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.NewLogger()
			defer log.Sync()
			ai.Init(log, args[0])
		},
	}
}
