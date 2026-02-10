package validate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/A-Flex-Box/cli/internal/meta"
)

// Result holds validation result.
type Result struct {
	OK     bool
	Err    error
	Item   *meta.HistoryItem
	Lang   string
	Exists bool
}

// ValidateFile checks file existence and optionally metadata.
func ValidateFile(filePath string, isAnswer bool, lang string) Result {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return Result{OK: false, Exists: false, Err: fmt.Errorf("file '%s' does not exist", filePath)}
	}

	if !isAnswer {
		return Result{OK: true, Exists: true}
	}

	if lang == "" {
		ext := filepath.Ext(filePath)
		if len(ext) > 1 {
			lang = ext[1:]
		} else {
			lang = "shell"
		}
	}

	item, err := meta.ParseMetadata(filePath, lang)
	if err != nil {
		return Result{OK: false, Exists: true, Err: err}
	}
	return Result{OK: true, Exists: true, Item: item, Lang: lang}
}
