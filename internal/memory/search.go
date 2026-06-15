package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Search performs a keyword search across INDEX.md and all subject files.
// Case-insensitive matching against: subject name, keywords column, and entry content.
// Results are sorted by score descending.
func (s *MemoryStore) Search(keyword string) ([]SearchResult, error) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return nil, nil
	}

	var results []SearchResult
	seen := make(map[string]bool) // dedup by subject+type+content

	// 1. Search index for subject name and keyword matches
	idx, err := s.ReadIndex()
	if err != nil {
		return nil, fmt.Errorf("search: read index: %w", err)
	}

	for _, si := range idx {
		subjectLower := strings.ToLower(si.Name)
		keywordsLower := strings.ToLower(si.Keywords)
		if strings.Contains(subjectLower, keyword) || strings.Contains(keywordsLower, keyword) {
			key := si.Name + "::index::" + keyword
			if !seen[key] {
				seen[key] = true
				results = append(results, SearchResult{
					Subject: si.Name,
					Type:    "index",
					Content: si.Summary,
					Tags:    si.Keywords,
					Score:   si.TopScore,
				})
			}
		}

		// 2. Search entry content within subject files
		entries, err := s.ReadSubject(si.Name)
		if err != nil {
			continue // skip unreadable files
		}
		for _, e := range entries {
			contentLower := strings.ToLower(e.Content)
			if !strings.Contains(contentLower, keyword) {
				continue
			}
			key := si.Name + "::" + e.Type + "::" + e.Content
			if seen[key] {
				continue
			}
			seen[key] = true

			// Score is a blend of entry score and whether content was matched
			matchBoost := 0.1
			score := e.Score*0.9 + matchBoost

			results = append(results, SearchResult{
				Subject: si.Name,
				Type:    e.Type,
				Content: truncateSummary(e.Content, 80),
				Tags:    strings.Join(e.Tags, ", "),
				Score:   score,
			})
		}
	}

	// 3. Also scan files not yet in index
	entries2, err := os.ReadDir(s.basePath)
	if err != nil {
		// Directory may not exist yet
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
		return results, nil
	}

	indexedNames := make(map[string]bool)
	for _, si := range idx {
		indexedNames[si.Name] = true
	}

	for _, entry := range entries2 {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") || name == "INDEX.md" || strings.HasSuffix(name, "-trash.md") {
			continue
		}
		subject := strings.TrimSuffix(name, ".md")
		if indexedNames[subject] {
			continue // already processed via index
		}
		subjectLower := strings.ToLower(subject)
		if !strings.Contains(subjectLower, keyword) {
			// Check file content
			data, err := os.ReadFile(filepath.Join(s.basePath, name))
			if err != nil {
				continue
			}
			if !strings.Contains(strings.ToLower(string(data)), keyword) {
				continue
			}
		}

		entries, err := s.ReadSubject(subject)
		if err != nil {
			continue
		}
		for _, e := range entries {
			contentLower := strings.ToLower(e.Content)
			if !strings.Contains(contentLower, keyword) {
				continue
			}
			key := subject + "::" + e.Type + "::" + e.Content
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, SearchResult{
				Subject: subject,
				Type:    e.Type,
				Content: truncateSummary(e.Content, 80),
				Tags:    strings.Join(e.Tags, ", "),
				Score:   e.Score,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}
