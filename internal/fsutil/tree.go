package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

// GenerateTree 生成项目目录结构的字符串表示
// 忽略 .git, bin, history/shell (为了保持 JSON 简洁)
func GenerateTree(root string) (string, error) {
	logger.Info("fsutil.GenerateTree start", logger.Context("params", map[string]any{"root": root})...)

	var sb strings.Builder
	sb.WriteString(".\n")
	skipped := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Debug("fsutil.GenerateTree walk error", zap.String("path", path), zap.Error(err))
			return err
		}

		if path == root {
			return nil
		}

		// 过滤规则
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "bin" {
				skipped++
				logger.Debug("fsutil.GenerateTree skip dir", zap.String("name", info.Name()))
				return filepath.SkipDir
			}
			// history 目录我们要，但是 history/shell 里面的脚本太多了，可以忽略内容只看目录
			if path == "history/shell" {
				skipped++
				logger.Debug("fsutil.GenerateTree skip history/shell content")
				// 记录目录本身，但跳过子内容
				indent := strings.Repeat("│   ", strings.Count(path, string(os.PathSeparator)))
				sb.WriteString(fmt.Sprintf("%s├── %s/ (archived scripts hidden)\n", indent, info.Name()))
				return filepath.SkipDir
			}
		}

		// 计算缩进
		relPath, _ := filepath.Rel(root, path)
		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("│   ", depth)
		
		marker := "├── "
		// 这里简化处理，不完美区分最后一个节点（└──），为了代码短小
		
		displayName := info.Name()
		if info.IsDir() {
			displayName += "/"
		}

		sb.WriteString(fmt.Sprintf("%s%s%s\n", indent, marker, displayName))
		return nil
	})

	result := sb.String()
	logger.Info("fsutil.GenerateTree done", logger.Context("result", map[string]any{
		"root": root, "output_len": len(result), "skipped_dirs": skipped,
	})...)
	return result, err
}
