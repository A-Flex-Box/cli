package memory

import (
	"os"
	"path/filepath"
)

// MemoryStore wraps a base path (~/.claude/memory/<repo>) for managing
// structured markdown memory files.
type MemoryStore struct {
	basePath string
}

// NewStore creates a MemoryStore for the given repository name.
// The base path resolves to ~/.claude/memory/<repo>.
func NewStore(repo string) *MemoryStore {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: use empty base; callers should check RepoExists().
		return &MemoryStore{basePath: ""}
	}
	return &MemoryStore{
		basePath: filepath.Join(home, ".claude", "memory", repo),
	}
}

// BasePath returns the resolved storage directory.
func (s *MemoryStore) BasePath() string {
	return s.basePath
}

// SubjectFilePath returns the path to a subject memory file.
func (s *MemoryStore) SubjectFilePath(subject string) string {
	return filepath.Join(s.basePath, subject+".md")
}

// SubjectTrashPath returns the path to a subject trash file.
func (s *MemoryStore) SubjectTrashPath(subject string) string {
	return filepath.Join(s.basePath, subject+"-trash.md")
}

// IndexPath returns the path to the index file.
func (s *MemoryStore) IndexPath() string {
	return filepath.Join(s.basePath, "INDEX.md")
}

// RepoExists returns true if the repo directory exists.
func (s *MemoryStore) RepoExists() bool {
	if s.basePath == "" {
		return false
	}
	info, err := os.Stat(s.basePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ensureDir creates the repo base directory if it doesn't exist.
func (s *MemoryStore) ensureDir() error {
	return os.MkdirAll(s.basePath, 0755)
}
