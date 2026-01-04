package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateTree 生成项目目录结构的字符串表示
// 忽略 .git, bin, history/shell (为了保持 JSON 简洁)
func GenerateTree(root string) (string, error) {
	var sb strings.Builder
	sb.WriteString(".\n")

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if path == root {
			return nil
		}

		// 过滤规则
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "bin" {
				return filepath.SkipDir
			}
			// history 目录我们要，但是 history/shell 里面的脚本太多了，可以忽略内容只看目录
			if path == "history/shell" {
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

	return sb.String(), err
}
