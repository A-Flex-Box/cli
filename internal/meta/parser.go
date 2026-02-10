package meta

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// FileChanges 记录文件变更详情
type FileChanges struct {
	Created   []string `json:"created,omitempty"`
	Modified  []string `json:"modified,omitempty"`
	Completed []string `json:"completed,omitempty"`
	Pending   []string `json:"pending,omitempty"`
}

// HistoryItem 对应 history.json 的结构
type HistoryItem struct {
	Timestamp       string            `json:"timestamp"`
	OriginalPrompt  string            `json:"original_prompt"`
	Summary         string            `json:"summary"`
	Action          string            `json:"action"`
	ExpectedOutcome string            `json:"expected_outcome"`
	// Iteration 迭代版本，用于操作迭代实现溯源 (e.g. v1.0.0)
	Iteration       string            `json:"iteration,omitempty"`
	// Context 存储 project_structure 等额外信息
	Context         map[string]string `json:"context,omitempty"`
	// FileChanges 记录文件变更详情
	FileChanges     *FileChanges      `json:"file_changes,omitempty"`
}

// LanguageConfig 定义不同语言的注释风格
type LanguageConfig struct {
	CommentPrefix string
}

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
	file, err := os.Open(filePath)
	if err != nil {
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
		return nil, fmt.Errorf("metadata incomplete (found %d fields)", foundFields)
	}

	return item, nil
}
