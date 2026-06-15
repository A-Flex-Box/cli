package memory

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/A-Flex-Box/cli/internal/memory"
	"github.com/spf13/cobra"
)

func newSaveCmd() *cobra.Command {
	var repo, subject, mtype, tags, title string
	var star bool
	var score float64

	cmd := &cobra.Command{
		Use:   "save [content]",
		Short: "Save a memory entry",
		Long: `Save a structured memory entry to the persistent store.

Content can be provided as a positional argument or read from stdin (use "-" as the argument).`,
		Example: `  cli memory save --repo dot --subject hypr --type findings "Super key triggers on release"
  echo "memory content" | cli memory save --repo dot --type ideas --tags "cli" -`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if repo == "" {
				fmt.Println("Error: --repo is required")
				os.Exit(1)
			}

			// Validate type
			allowedTypes := map[string]bool{
				"findings": true, "hypotheses": true, "questions": true,
				"tools": true, "ideas": true, "mistakes": true,
			}
			if !allowedTypes[mtype] {
				fmt.Printf("Error: invalid type %q; must be one of: findings, hypotheses, questions, tools, ideas, mistakes\n", mtype)
				os.Exit(1)
			}

			// Get content
			var content string
			if len(args) > 0 && args[0] == "-" {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					fmt.Printf("Error: reading stdin: %v\n", err)
					os.Exit(1)
				}
				content = strings.TrimSpace(string(data))
			} else if len(args) > 0 {
				content = args[0]
			} else {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					fmt.Printf("Error: reading stdin: %v\n", err)
					os.Exit(1)
				}
				content = strings.TrimSpace(string(data))
			}

			if content == "" {
				fmt.Println("Error: content cannot be empty")
				os.Exit(1)
			}

			// Auto-generate subject from first 60 chars if empty
			if subject == "" {
				subject = truncateSubject(content, 60)
			}

			// Parse tags
			var tagList []string
			if tags != "" {
				for _, t := range strings.Split(tags, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						tagList = append(tagList, t)
					}
				}
			}

			store := memory.NewStore(repo)

			// Ensure repo exists
			if err := store.InitRepo(); err != nil {
				fmt.Printf("Error: initializing repo: %v\n", err)
				os.Exit(1)
			}

			entry := memory.Entry{
				Content: content,
				Type:    mtype,
				Tags:    tagList,
				Starred: star,
			}

			if score > 0 {
				entry.Score = score
				entry.Starred = true
			}

			if title != "" {
				entry.Summary = title
			}

			if err := store.AppendEntry(subject, entry); err != nil {
				fmt.Printf("Error: saving memory: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✅ Memory saved to %s\n", subject)
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "Repository identifier (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject/title (auto-generated from content if omitted)")
	cmd.Flags().StringVar(&mtype, "type", "findings", "Type: findings, hypotheses, questions, tools, ideas, mistakes")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().BoolVar(&star, "star", false, "Auto-star the new entry")
	cmd.Flags().Float64Var(&score, "score", 0.0, "AI attention weight (0.0-1.0); if set, the entry is also starred")
	cmd.Flags().StringVar(&title, "title", "", "Short title for the entry (affects INDEX summary)")

	return cmd
}

func truncateSubject(s string, maxLen int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen])
}
