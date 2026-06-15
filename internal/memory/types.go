package memory

import "time"

// Entry represents a single memory entry within a subject file.
type Entry struct {
	ID         int       // 1-based auto-increment within file
	Timestamp  time.Time
	Content    string
	Tags       []string
	Score      float64    // attention score (0.0-1.0)
	Type       string     // findings | hypotheses | questions | tools | ideas | mistakes
	Summary    string     // short title/summary for INDEX; auto-generated from Content if empty
	Starred    bool
	LastAccess time.Time
	LoadCount  int
}

// SubjectIndex represents one row in INDEX.md.
type SubjectIndex struct {
	Name       string
	Keywords   string
	Entries    int
	TopScore   float64
	LastAccess string // ISO date
	Summary    string
}

// SearchResult represents a search hit.
type SearchResult struct {
	Subject string
	Type    string
	Content string
	Tags    string
	Score   float64
}
