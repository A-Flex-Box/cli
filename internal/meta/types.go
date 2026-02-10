package meta

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
	Iteration       string            `json:"iteration,omitempty"`
	Context         map[string]string `json:"context,omitempty"`
	FileChanges     *FileChanges      `json:"file_changes,omitempty"`
}

// LanguageConfig 定义不同语言的注释风格
type LanguageConfig struct {
	CommentPrefix string
}
