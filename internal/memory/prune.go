package memory

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Prune keeps the top N entries by score in a subject file and moves the
// rest to <subject>-trash.md with a [Trashed YYYY-MM-DD] marker.
// Default maxEntries is 50 if max <= 0.
// Returns the count of entries moved to trash.
func (s *MemoryStore) Prune(subject string, maxEntries int) (int, error) {
	if maxEntries <= 0 {
		maxEntries = 50
	}

	entries, err := s.ReadSubject(subject)
	if err != nil {
		return 0, fmt.Errorf("prune: read subject %q: %w", subject, err)
	}

	if len(entries) <= maxEntries {
		return 0, nil // nothing to prune
	}

	// Sort by score descending
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	keep := sorted[:maxEntries]
	trash := sorted[maxEntries:]

	// Re-assign IDs for kept entries
	for i := range keep {
		keep[i].ID = i + 1
	}

	// Write the kept entries back
	if err := s.WriteSubject(subject, keep); err != nil {
		return 0, fmt.Errorf("prune: write subject %q: %w", subject, err)
	}

	// Append trashed entries to <subject>-trash.md
	trashPath := s.SubjectTrashPath(subject)
	existingTrash := readTrashFile(trashPath)
	allTrash := append(trash, existingTrash...)

	trashDate := time.Now().Format("2006-01-02")
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s Trash\n\n", subject))
	for _, e := range allTrash {
		dateStr := e.Timestamp.Format("2006-01-02")
		metaParts := []string{
			fmt.Sprintf("score:%.2f", e.Score),
			fmt.Sprintf("[Trashed %s]", trashDate),
		}
		if len(e.Tags) > 0 {
			metaParts = append(metaParts, "tags:"+strings.Join(e.Tags, ","))
		}
		meta := strings.Join(metaParts, " | ")

		b.WriteString(fmt.Sprintf("- %s | %s\n", dateStr, meta))
		contentLines := strings.Split(e.Content, "\n")
		for _, cl := range contentLines {
			b.WriteString(fmt.Sprintf("  %s\n", cl))
		}
	}

	if err := os.WriteFile(trashPath, []byte(b.String()), 0644); err != nil {
		return 0, fmt.Errorf("prune: write trash %q: %w", subject, err)
	}

	// Update index with new entry count and top score
	idx, err := s.ReadIndex()
	if err != nil {
		return 0, fmt.Errorf("prune: read index: %w", err)
	}
	topScore := 0.0
	for _, e := range keep {
		if e.Score > topScore {
			topScore = e.Score
		}
	}
	for i, si := range idx {
		if si.Name == subject {
			idx[i].Entries = len(keep)
			idx[i].TopScore = topScore
			idx[i].LastAccess = time.Now().Format("2006-01-02")
			break
		}
	}
	if err := s.WriteIndex(idx); err != nil {
		return 0, fmt.Errorf("prune: update index: %w", err)
	}

	return len(trash), nil
}

// readTrashFile reads the trash file if it exists, returning any entries found.
func readTrashFile(path string) []Entry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var entries []Entry
	id := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			id++
			rest := strings.TrimPrefix(trimmed, "- ")
			parts := strings.SplitN(rest, " | ", 2)
			if len(parts) < 2 {
				continue
			}
			entries = append(entries, Entry{
				ID:      id,
				Content: parts[len(parts)-1],
				Score:   0,
				Type:    "findings",
			})
		}
	}
	return entries
}
