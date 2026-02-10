package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/A-Flex-Box/cli/internal/fsutil"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/A-Flex-Box/cli/internal/meta"
	"go.uber.org/zap"
)

// Add extracts metadata from file, captures project structure, and appends to history.json.
func Add(historyPath string, filePath string) error {
	logger.Info("history.Add start", logger.Context("params", map[string]any{
		"history_path": historyPath, "file_path": filePath,
	})...)

	lang := "shell"
	if ext := filepath.Ext(filePath); len(ext) > 1 {
		lang = ext[1:]
	}
	logger.Debug("history.Add inferred lang", zap.String("lang", lang))

	newItem, err := meta.ParseMetadata(filePath, lang)
	if err != nil {
		logger.Warn("history.Add parse metadata failed", zap.Error(err), zap.String("file_path", filePath), zap.String("lang", lang))
		return fmt.Errorf("failed to extract metadata: %w", err)
	}
	logger.Info("history.Add metadata parsed", logger.Context("item", map[string]any{
		"timestamp": newItem.Timestamp, "action": newItem.Action, "iteration": newItem.Iteration,
	})...)

	treeStr, err := fsutil.GenerateTree(".")
	if err != nil {
		logger.Warn("history.Add generate tree failed", zap.Error(err))
		return fmt.Errorf("failed to generate project structure: %w", err)
	}
	logger.Debug("history.Add tree generated", zap.Int("tree_len", len(treeStr)))
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
		logger.Warn("history.Add marshal failed", zap.Error(err))
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(historyPath, newData, 0644); err != nil {
		logger.Warn("history.Add write failed", zap.Error(err), zap.String("history_path", historyPath))
		return fmt.Errorf("failed to write history: %w", err)
	}
	logger.Info("history.Add done", logger.Context("result", map[string]any{
		"history_path": historyPath, "file_path": filePath, "items_count": len(items),
	})...)
	return nil
}
