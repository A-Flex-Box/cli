package memory

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// sectionHeaders maps Entry.Type to the markdown header used in subject files.
var sectionHeaders = map[string]string{
	"findings":    "## Findings",
	"hypotheses":  "## Hypotheses",
	"questions":   "## Open Questions",
	"tools":       "## Tools & Techniques",
	"ideas":       "## Ideas",
	"mistakes":    "## Mistakes / Lessons",
}

// headerSection is the reverse map.
var headerSection = func() map[string]string {
	m := make(map[string]string, len(sectionHeaders))
	for k, v := range sectionHeaders {
		m[v] = k
	}
	return m
}()

// InitRepo creates the repo directory and writes an initial INDEX.md if it
// doesn't already exist. Idempotent.
func (s *MemoryStore) InitRepo() error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("init repo: %w", err)
	}
	idxPath := s.IndexPath()
	if _, err := os.Stat(idxPath); err == nil {
		return nil // already exists
	}
	idx := fmt.Sprintf("# Memory Index\n\n## %s\n\n| subject | keywords | entries | top_attention | last_accessed | summary |\n|---------|----------|---------|---------------|---------------|---------|\n", filepath.Base(s.basePath))
	if err := os.WriteFile(idxPath, []byte(idx), 0644); err != nil {
		return fmt.Errorf("init repo: write index: %w", err)
	}
	return nil
}

// ReadSubject reads a subject memory file and parses it into entries.
// Returns empty slice (no error) if the file does not exist.
func (s *MemoryStore) ReadSubject(subject string) ([]Entry, error) {
	path := s.SubjectFilePath(subject)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read subject %q: %w", subject, err)
	}
	return parseSubjectFile(subject, string(data))
}

// parseSubjectFile parses a subject markdown file into entries.
func parseSubjectFile(subject, content string) ([]Entry, error) {
	lines := strings.Split(content, "\n")
	var entries []Entry
	var currentSection string
	entryID := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section header
		if strings.HasPrefix(trimmed, "## ") {
			section := strings.TrimPrefix(trimmed, "## ")
			// Normalize section name
			if sec, ok := headerSection["## "+section]; ok {
				currentSection = sec
			} else {
				// Try matching raw header
				matched := false
				for header, sec := range headerSection {
					if strings.EqualFold(header, "## "+section) {
						currentSection = sec
						matched = true
						break
					}
				}
				if !matched {
					currentSection = ""
				}
			}
			continue
		}

		// Parse numbered entry: "1. 2026-06-12 | score:0.92 | tags:a,b\n   Content line"
		if currentSection != "" && strings.TrimSpace(line) != "" {
			if entry, ok := parseEntryLine(trimmed, currentSection, &entryID); ok {
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

// parseEntryLine parses a single entry line like:
//
//	1. 2026-06-12 | score:0.92 | tags:keybinding,debug
//	   Content line here
func parseEntryLine(line, section string, idPtr *int) (Entry, bool) {
	// Entry starts with a number followed by a period
	if len(line) < 2 || line[0] < '0' || line[0] > '9' {
		return Entry{}, false
	}

	// Find the period after the number
	dotIdx := strings.Index(line, ". ")
	if dotIdx < 0 {
		return Entry{}, false
	}

	rest := strings.TrimSpace(line[dotIdx+2:])
	if rest == "" {
		return Entry{}, false
	}

	*idPtr++

	e := Entry{
		ID:         *idPtr,
		Type:       section,
		Starred:    false,
		Score:      0.5,
		LastAccess: time.Now(),
		LoadCount:  1,
	}

	// Extract date: first token before space or |
	tokens := strings.SplitN(rest, " | ", 3)
	dateStr := strings.TrimSpace(tokens[0])

	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		e.Timestamp = t
	} else {
		// Date field might be missing; treat rest as partial
		e.Timestamp = time.Now()
		// Re-parse from beginning
		if len(tokens) == 1 {
			e.Content = tokens[0]
			return e, true
		}
	}

	// Parse metadata and content
	if len(tokens) >= 2 {
		metaPart := tokens[1]
		// Parse score:tag pairs
		for _, part := range strings.Split(metaPart, "|") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "score:") {
				scoreStr := strings.TrimPrefix(part, "score:")
				if score, err := strconv.ParseFloat(scoreStr, 64); err == nil {
					e.Score = math.Max(0, math.Min(1, score))
				}
			} else if strings.HasPrefix(part, "tags:") {
				tagStr := strings.TrimPrefix(part, "tags:")
				e.Tags = strings.Split(tagStr, ",")
				for i := range e.Tags {
					e.Tags[i] = strings.TrimSpace(e.Tags[i])
				}
			}
		}
	}

	if len(tokens) >= 3 {
		e.Content = strings.TrimSpace(tokens[2])
	} else if len(tokens) == 2 {
		// No metadata, content is after date
		e.Content = strings.TrimSpace(tokens[1])
	} else {
		e.Content = strings.TrimSpace(tokens[0])
	}

	return e, true
}

// WriteSubject writes a subject memory file from entries.
func (s *MemoryStore) WriteSubject(subject string, entries []Entry) error {
	path := s.SubjectFilePath(subject)
	content := renderSubjectFile(subject, entries)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write subject %q: %w", subject, err)
	}
	return nil
}

// renderSubjectFile renders entries back into markdown format.
func renderSubjectFile(subject string, entries []Entry) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s Memory\n\n", subject))

	// Group entries by type in section order
	sectionOrder := []string{"findings", "hypotheses", "questions", "tools", "ideas", "mistakes"}
	grouped := make(map[string][]Entry)
	for _, e := range entries {
		grouped[e.Type] = append(grouped[e.Type], e)
	}

	for _, sec := range sectionOrder {
		header, ok := sectionHeaders[sec]
		if !ok {
			continue
		}
		b.WriteString(header + "\n\n")
		secEntries := grouped[sec]
		// Sort by ID ascending (or timestamp descending — spec says newest first)
		sort.Slice(secEntries, func(i, j int) bool {
			return secEntries[i].Timestamp.After(secEntries[j].Timestamp)
		})
		for _, e := range secEntries {
			renderEntry(&b, e)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderEntry writes a single entry in markdown format.
func renderEntry(b *strings.Builder, e Entry) {
	dateStr := e.Timestamp.Format("2006-01-02")
	metaParts := []string{
		fmt.Sprintf("score:%.2f", e.Score),
	}
	if len(e.Tags) > 0 {
		metaParts = append(metaParts, "tags:"+strings.Join(e.Tags, ","))
	}
	meta := strings.Join(metaParts, " | ")

	// Split content into lines for indentation
	contentLines := strings.Split(e.Content, "\n")
	firstLine := contentLines[0]
	_ = firstLine

	b.WriteString(fmt.Sprintf("%d. %s | %s\n", e.ID, dateStr, meta))
	for _, cl := range contentLines {
		b.WriteString(fmt.Sprintf("   %s\n", cl))
	}
}

// ReadIndex reads INDEX.md and parses the table rows.
// Returns empty slice (no error) if the file does not exist.
func (s *MemoryStore) ReadIndex() ([]SubjectIndex, error) {
	path := s.IndexPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SubjectIndex{}, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}
	return parseIndex(string(data))
}

// parseIndex parses INDEX.md content into SubjectIndex entries.
func parseIndex(content string) ([]SubjectIndex, error) {
	lines := strings.Split(content, "\n")
	var results []SubjectIndex
	inTable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect table separator row: |---|---...
		if strings.HasPrefix(trimmed, "|---") {
			inTable = true
			continue
		}

		if !inTable {
			continue
		}

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Parse table row: | subject | keywords | entries | top_attention | last_accessed | summary |
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}

		// Strip leading/trailing |
		inner := strings.TrimPrefix(trimmed, "|")
		inner = strings.TrimSuffix(inner, "|")
		cells := splitPipeRow(inner)
		if len(cells) < 6 {
			continue
		}

		entries, _ := strconv.Atoi(strings.TrimSpace(cells[2]))
		topScore, _ := strconv.ParseFloat(strings.TrimSpace(cells[3]), 64)

		results = append(results, SubjectIndex{
			Name:       strings.TrimSpace(cells[0]),
			Keywords:   strings.TrimSpace(cells[1]),
			Entries:    entries,
			TopScore:   topScore,
			LastAccess: strings.TrimSpace(cells[4]),
			Summary:    strings.TrimSpace(cells[5]),
		})
	}

	return results, nil
}

// splitPipeRow splits a pipe-delimited table row respecting cell boundaries.
func splitPipeRow(row string) []string {
	parts := strings.Split(row, "|")
	var result []string
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

// WriteIndex writes INDEX.md from SubjectIndex entries.
func (s *MemoryStore) WriteIndex(index []SubjectIndex) error {
	repoName := filepath.Base(s.basePath)
	var b strings.Builder
	b.WriteString("# Memory Index\n\n")
	b.WriteString(fmt.Sprintf("## %s\n\n", repoName))
	b.WriteString("| subject | keywords | entries | top_attention | last_accessed | summary |\n")
	b.WriteString("|---------|----------|---------|---------------|---------------|---------|\n")

	sort.Slice(index, func(i, j int) bool {
		return index[i].Name < index[j].Name
	})
	for _, si := range index {
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %.2f | %s | %s |\n",
			si.Name, si.Keywords, si.Entries, si.TopScore, si.LastAccess, si.Summary))
	}

	path := s.IndexPath()
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

// AppendEntry appends an entry to a subject file and updates the index.
// Creates the subject file with a template if it doesn't exist.
func (s *MemoryStore) AppendEntry(subject string, entry Entry) error {
	// Read existing entries
	entries, err := s.ReadSubject(subject)
	if err != nil {
		return fmt.Errorf("append entry: %w", err)
	}

	// Assign ID
	maxID := 0
	for _, e := range entries {
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	entry.ID = maxID + 1
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.Score <= 0 {
		entry.Score = 0.5
	}
	entry.LastAccess = time.Now()
	entry.LoadCount = 1

	// Prepend so newest is first
	entries = append([]Entry{entry}, entries...)

	// Write subject file
	if err := s.WriteSubject(subject, entries); err != nil {
		return fmt.Errorf("append entry: %w", err)
	}

	// Update index
	idx, err := s.ReadIndex()
	if err != nil {
		return fmt.Errorf("append entry: read index: %w", err)
	}

	found := false
	topScore := entry.Score
	for _, e := range entries {
		if e.Score > topScore {
			topScore = e.Score
		}
	}

	for i, si := range idx {
		if si.Name == subject {
			idx[i].Entries = len(entries)
			idx[i].TopScore = topScore
			idx[i].LastAccess = time.Now().Format("2006-01-02")
			found = true
			break
		}
	}

	if !found {
		idx = append(idx, SubjectIndex{
			Name:       subject,
			Keywords:   strings.Join(entry.Tags, ", "),
			Entries:    len(entries),
			TopScore:   topScore,
			LastAccess: time.Now().Format("2006-01-02"),
			Summary: func() string {
				if entry.Summary != "" {
					return entry.Summary
				}
				return truncateSummary(entry.Content, 60)
			}(),
		})
	}

	return s.WriteIndex(idx)
}

// truncateSummary truncates a string to maxLen chars for index summaries.
func truncateSummary(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
