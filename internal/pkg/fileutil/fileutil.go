package fileutil

import (
	"os"
	"strings"
)

// ReadFileTrim reads a file and returns its contents trimmed of whitespace.
func ReadFileTrim(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// FileExists returns true if the path exists and is accessible.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
