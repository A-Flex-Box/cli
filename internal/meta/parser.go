package meta

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

var langConfigs = map[string]LanguageConfig{
	"shell":  {"#"},
	"sh":     {"#"},
	"bash":   {"#"},
	"py":     {"#"},
	"go":     {"//"},
	"cpp":    {"//"},
}

// ParseMetadata 读取文件并提取 Metadata Block
func ParseMetadata(filePath string, lang string) (*HistoryItem, error) {
	logger.Info("meta.ParseMetadata start", logger.Context("params", map[string]any{"file_path": filePath, "lang": lang})...)
	file, err := os.Open(filePath)
	if err != nil {
		logger.Warn("meta.ParseMetadata open failed", zap.Error(err), zap.String("file_path", filePath))
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	config, ok := langConfigs[strings.ToLower(lang)]
	if !ok {
		config = LanguageConfig{CommentPrefix: "#"}
	}
	prefix := config.CommentPrefix

	scanner := bufio.NewScanner(file)
	inBlock := false
	item := &HistoryItem{
		Context: make(map[string]string),
	}
	foundFields := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.Contains(line, "METADATA_START") {
			inBlock = true
			continue
		}
		if strings.Contains(line, "METADATA_END") {
			break
		}

		if inBlock {
			content := strings.TrimPrefix(line, prefix)
			content = strings.TrimSpace(content)

			parts := strings.SplitN(content, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(strings.ToLower(parts[0]))
				val := strings.TrimSpace(parts[1])

				switch key {
				case "timestamp":
					item.Timestamp = val
					foundFields++
				case "original_prompt":
					item.OriginalPrompt = val
					foundFields++
				case "summary":
					item.Summary = val
					foundFields++
				case "action":
					item.Action = val
					foundFields++
				case "expected_outcome":
					item.ExpectedOutcome = val
					foundFields++
				case "iteration":
					item.Iteration = strings.TrimSpace(val)
				// 其他字段可以放入 Context，暂时只解析核心字段
				}
			}
		}
	}

	if foundFields < 3 {
		logger.Warn("meta.ParseMetadata incomplete", zap.Int("found_fields", foundFields), zap.String("file_path", filePath))
		return nil, fmt.Errorf("metadata incomplete (found %d fields)", foundFields)
	}

	logger.Info("meta.ParseMetadata done", logger.Context("result", map[string]any{
		"file_path": filePath, "found_fields": foundFields,
		"timestamp": item.Timestamp, "action": item.Action, "iteration": item.Iteration,
	})...)
	return item, nil
}
