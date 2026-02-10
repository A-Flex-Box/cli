package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/A-Flex-Box/cli/internal/fsutil"
)

// HistoryItem for prompt context (minimal struct).
type HistoryItem struct {
	Timestamp       string            `json:"timestamp"`
	OriginalPrompt  string            `json:"original_prompt"`
	Summary         string            `json:"summary"`
	Action          string            `json:"action"`
	ExpectedOutcome string            `json:"expected_outcome"`
	Context         map[string]string `json:"context,omitempty"`
}

// BuildContextPrompt generates project context from history.
func BuildContextPrompt(historyPath string) string {
	var items []HistoryItem
	if data, err := os.ReadFile(historyPath); err == nil && len(data) > 0 {
		json.Unmarshal(data, &items)
	}

	var sb strings.Builder
	sb.WriteString("# Project Context (History)\n")

	var lastStructure string
	for _, item := range items {
		if val, ok := item.Context["project_structure"]; ok && val != "" {
			lastStructure = val
		}
	}

	if lastStructure == "" {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: Project structure missing from history. Using real-time file system snapshot instead.\n")
		if liveTree, err := fsutil.GenerateTree("."); err == nil {
			lastStructure = liveTree
		}
	}

	sb.WriteString("Recent development steps for context:\n\n")
	startIdx := 0
	if len(items) > 3 {
		startIdx = len(items) - 3
	}

	for i := startIdx; i < len(items); i++ {
		item := items[i]
		sb.WriteString(fmt.Sprintf("## History Step %d (%s)\n", i+1, item.Timestamp))
		shortPrompt := strings.ReplaceAll(item.OriginalPrompt, "\n", " ")
		if len(shortPrompt) > 80 {
			shortPrompt = shortPrompt[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("- Request: %s\n", shortPrompt))
		sb.WriteString(fmt.Sprintf("- Action: %s\n\n", item.Action))
	}

	if lastStructure != "" {
		sb.WriteString("## Current Project Structure\n")
		sb.WriteString("```text\n")
		sb.WriteString(lastStructure)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("--------------------------------------------------\n")
	return sb.String()
}

// AppendOutputConstraints appends format constraints to the prompt.
func AppendOutputConstraints(sb *strings.Builder, requirement, outputFormat string) {
	if outputFormat != "" && outputFormat != "text" {
		sb.WriteString("## Output Format Constraints\n")
		sb.WriteString(fmt.Sprintf("1. Provide the solution as a **single %s file**.\n", outputFormat))
		sb.WriteString("2. **CRITICAL: METADATA HEADER REQUIRED**\n")
		sb.WriteString("   The file MUST start with a metadata header block in comments.\n")
		sb.WriteString("   The `original_prompt` field MUST contain the **EXACT FULL TEXT** of the 'New Requirement' section below. **DO NOT TRUNCATE.**\n")

		commentChar := "#"
		if outputFormat == "go" || outputFormat == "cpp" {
			commentChar = "//"
		}
		sb.WriteString(fmt.Sprintf("   Format: %s METADATA_START ... %s METADATA_END\n\n", commentChar, commentChar))
	}
}

// GenerateTaskPrompt generates a task prompt with context.
func GenerateTaskPrompt(historyPath string, requirement, outputFormat string) string {
	var sb strings.Builder
	sb.WriteString(BuildContextPrompt(historyPath))
	sb.WriteString("# New Requirement (Current Task)\n")
	sb.WriteString(requirement)
	sb.WriteString("\n\n")
	AppendOutputConstraints(&sb, requirement, outputFormat)
	return sb.String()
}

// GenerateCommitPrompt generates a commit message prompt from git diff.
func GenerateCommitPrompt(historyPath string, optionalInstruction string) (string, error) {
	diffCmd := exec.Command("git", "diff", "HEAD")
	diffOut, err := diffCmd.CombinedOutput()
	diffStr := string(diffOut)
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	if len(strings.TrimSpace(diffStr)) == 0 {
		return "", fmt.Errorf("no changes detected (git diff is empty)")
	}

	var sb strings.Builder
	sb.WriteString("# Task: Generate Git Commit Message\n")
	sb.WriteString("You are a Senior Developer. Please write a semantic git commit message for the following code changes.\n\n")
	sb.WriteString("## Code Changes (Git Diff)\n")
	sb.WriteString("```diff\n")
	if len(diffStr) > 8000 {
		sb.WriteString(diffStr[:8000] + "\n... (diff truncated) ...")
	} else {
		sb.WriteString(diffStr)
	}
	sb.WriteString("\n```\n\n")
	sb.WriteString(BuildContextPrompt(historyPath))
	sb.WriteString("## Instruction\n")
	instruction := "Analyze the diff above. Generate a concise and meaningful commit message following **Conventional Commits** format."
	if optionalInstruction != "" {
		instruction = optionalInstruction
	}
	sb.WriteString(instruction)
	sb.WriteString("\n\n")
	sb.WriteString("## Expected Output Format\n")
	sb.WriteString("```text\n")
	sb.WriteString("<type>(<scope>): <subject>\n\n")
	sb.WriteString("<body>\n\n")
	sb.WriteString("[Optional Footer: Ref #IssueID]\n")
	sb.WriteString("```\n")
	return sb.String(), nil
}
