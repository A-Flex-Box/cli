package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

// Init creates standard AI project directory structure.
func Init(log *zap.Logger, projectName string) {
	if log == nil {
		log = logger.NewLogger()
		defer log.Sync()
	}
	log.Info("初始化AI项目", zap.String("project_name", projectName))
	structure := map[string]string{
		"data/raw":       "原始不可变数据",
		"data/processed": "清洗后的特征数据",
		"models":         "模型权重 checkpoints",
		"notebooks":      "Jupyter Notebooks",
		"src":            "源代码",
		"src/utils":      "工具函数",
		"logs":           "Training Logs",
		"configs":        "Hyperparameters",
	}
	fmt.Printf("🏗  Initializing Project: %s\n", projectName)
	for path, desc := range structure {
		fullPath := filepath.Join(projectName, path)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			log.Error("创建目录失败", zap.String("path", fullPath), zap.Error(err))
			continue
		}
		readmePath := filepath.Join(fullPath, "README.md")
		if err := os.WriteFile(readmePath, []byte(desc), 0644); err != nil {
			log.Error("创建README失败", zap.String("path", readmePath), zap.Error(err))
		} else {
			log.Info("创建目录和README", zap.String("path", fullPath))
		}
	}
	fmt.Println("✅ Done.")
	log.Info("项目初始化完成", zap.String("project_name", projectName))
}
