package memory

import (
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/memory"
	"github.com/spf13/cobra"
)

func newLoadCmd() *cobra.Command {
	var repo, subject string

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load a memory entry or show repo index",
		Long: `Load and display a memory entry by subject, or show the repo index if --subject is omitted.`,
		Example: `  cli memory load --repo dot
  cli memory load --repo dot --subject hypr`,
		Run: func(cmd *cobra.Command, args []string) {
			if repo == "" {
				fmt.Println("Error: --repo is required")
				os.Exit(1)
			}

			store := memory.NewStore(repo)

			if !store.RepoExists() {
				fmt.Printf("Error: repo %q not found\n", repo)
				os.Exit(1)
			}

			if subject != "" {
				entries, err := store.ReadSubject(subject)
				if err != nil {
					fmt.Printf("Error: reading subject %q: %v\n", subject, err)
					os.Exit(1)
				}
				if len(entries) == 0 {
					fmt.Printf("No entries found for subject %q\n", subject)
					return
				}

				// Print as raw markdown (read the file directly)
				path := store.SubjectFilePath(subject)
				data, err := os.ReadFile(path)
				if err != nil {
					fmt.Printf("Error: reading file: %v\n", err)
					os.Exit(1)
				}
				fmt.Print(string(data))
			} else {
				// Show INDEX table
				idx, err := store.ReadIndex()
				if err != nil {
					fmt.Printf("Error: reading index: %v\n", err)
					os.Exit(1)
				}
				if len(idx) == 0 {
					fmt.Printf("No memory entries for repo %q\n", repo)
					return
				}
				printIndex(repo, idx)
			}
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "Repository identifier (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject to load (omit to show repo index)")

	return cmd
}

// printIndex displays a SubjectIndex slice as a human-readable table.
func printIndex(repo string, idx []memory.SubjectIndex) {
	numSubjects := len(idx)

	fmt.Printf(" Memory for repo %q (%d subject%s)\n\n", repo, numSubjects, pluralS(numSubjects))
	fmt.Printf("  %-12s | %-26s | %-7s | %-5s | %-13s | %s\n", "subject", "keywords", "entries", "score", "last_accessed", "summary")
	fmt.Printf("  %s-|-%s-|-%s-|-%s-|-%s-|-%s\n",
		repeat("-", 12), repeat("-", 26), repeat("-", 7), repeat("-", 5), repeat("-", 13), repeat("-", 0))

	for _, si := range idx {
		summary := truncateDisplay(si.Summary, 40)
		fmt.Printf("  %-12s | %-26s | %-7d | %-5.2f | %-13s | %s\n",
			si.Name, si.Keywords, si.Entries, si.TopScore, si.LastAccess, summary)
	}
}

func pluralS(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// truncateDisplay truncates a string to maxLen runes, appending "..." if truncated.
func truncateDisplay(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
