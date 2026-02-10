package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/A-Flex-Box/cli/internal/fsutil"
	"github.com/A-Flex-Box/cli/internal/meta"
)

// Add extracts metadata from file, captures project structure, and appends to history.json.
func Add(historyPath string, filePath string) error {
	lang := "shell"
	if ext := filepath.Ext(filePath); len(ext) > 1 {
		lang = ext[1:]
	}

	newItem, err := meta.ParseMetadata(filePath, lang)
	if err != nil {
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	treeStr, err := fsutil.GenerateTree(".")
	if err != nil {
		return fmt.Errorf("failed to generate project structure: %w", err)
	}
	if newItem.Context == nil {
		newItem.Context = make(map[string]string)
	}
	newItem.Context["project_structure"] = treeStr

	var items []meta.HistoryItem
	if _, err := os.Stat(historyPath); err == nil {
		data, err := os.ReadFile(historyPath)
		if err != nil {
			return fmt.Errorf("failed to read history: %w", err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to parse history.json: %w", err)
			}
		}
	}

	// Backfill iteration for items that lack it
	for i := range items {
		if items[i].Iteration == "" {
			items[i].Iteration = "v1.0.0"
		}
	}

	// Use iteration from metadata, or default
	if newItem.Iteration == "" {
		newItem.Iteration = "v1.0.0"
	}
	items = append(items, *newItem)

	newData, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(historyPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write history: %w", err)
	}
	return nil
}
