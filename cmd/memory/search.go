package memory

import (
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/memory"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var repo, keyword string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search memory entries by keyword",
		Long:  `Search across all subjects in a repo for entries matching the given keyword.`,
		Example: `  cli memory search --repo dot --keyword "keybinding"`,
		Run: func(cmd *cobra.Command, args []string) {
			if repo == "" {
				fmt.Println("Error: --repo is required")
				os.Exit(1)
			}

			if keyword == "" {
				fmt.Println("Error: --keyword is required")
				os.Exit(1)
			}

			store := memory.NewStore(repo)

			results, err := store.Search(keyword)
			if err != nil {
				fmt.Printf("Error: searching memory: %v\n", err)
				os.Exit(1)
			}

			if len(results) == 0 {
				fmt.Printf("No results for %q in repo %q\n", keyword, repo)
				return
			}

			// Group results by subject and display
			fmt.Printf(" Search results for %q in repo %q (%d match%s)\n\n", keyword, repo, len(results), pluralS(len(results)))
			for _, r := range results {
				star := ""
				if r.Score > 0.8 {
					star = " ⭐"
				}
				fmt.Printf("  [%s]%s %s\n", r.Type, star, r.Content)
				fmt.Printf("        subject: %s  tags: %s  score: %.2f\n", r.Subject, r.Tags, r.Score)
			}
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "Repository identifier (required)")
	cmd.Flags().StringVar(&keyword, "keyword", "", "Keyword to search for (required)")

	return cmd
}
