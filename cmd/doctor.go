package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "检查环境健康状态",
	Long:  `检查 Git、Go 以及项目配置文件是否存在且正常。`,
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		defer log.Sync()

		log.Info("开始环境健康检查", zap.String("os", runtime.GOOS), zap.String("arch", runtime.GOARCH))
		fmt.Printf("🏥 CLI Doctor Report (%s/%s)\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println("--------------------------------------------------")

		// 检查 Go
		if path, err := exec.LookPath("go"); err == nil {
			fmt.Printf("✅ Go installed: %s\n", path)
			log.Info("Go已安装", zap.String("path", path))
		} else {
			fmt.Printf("❌ Go NOT found!\n")
			log.Error("Go未找到", zap.Error(err))
		}

		// 检查 Git
		if path, err := exec.LookPath("git"); err == nil {
			fmt.Printf("✅ Git installed: %s\n", path)
			log.Info("Git已安装", zap.String("path", path))
		} else {
			fmt.Printf("❌ Git NOT found!\n")
			log.Error("Git未找到", zap.Error(err))
		}

		// 检查 Make
		if path, err := exec.LookPath("make"); err == nil {
			fmt.Printf("✅ Make installed: %s\n", path)
			log.Info("Make已安装", zap.String("path", path))
		} else {
			fmt.Printf("❌ Make NOT found!\n")
			log.Error("Make未找到", zap.Error(err))
		}

		// 检查 History
		if _, err := exec.LookPath("history/history.json"); err != nil {
			fmt.Printf("✅ History database found.\n")
			log.Info("历史数据库已找到")
		} else {
			log.Warn("历史数据库未找到")
		}

		fmt.Println("--------------------------------------------------")
		fmt.Println("Diagnosis complete.")
		log.Info("环境健康检查完成")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
