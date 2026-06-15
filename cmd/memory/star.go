package memory

import (
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/memory"
	"github.com/spf13/cobra"
)

func newStarCmd() *cobra.Command {
	var repo, subject string
	var id int
	var score float64

	cmd := &cobra.Command{
		Use:   "star",
		Short: "Star or rate a memory entry",
		Long: `Set the AI weight score for an entry, or toggle the starred flag.

If --score is provided, set the entry's score to that value.
If --score is omitted (0.0), toggle the starred flag.`,
		Example: `  cli memory star --repo dot --subject hypr --id 1 --score 0.9
  cli memory star --repo dot --subject hypr --id 1`,
		Run: func(cmd *cobra.Command, args []string) {
			if repo == "" {
				fmt.Println("Error: --repo is required")
				os.Exit(1)
			}

			if subject == "" {
				fmt.Println("Error: --subject is required")
				os.Exit(1)
			}

			store := memory.NewStore(repo)

			if !store.RepoExists() {
				fmt.Printf("Error: repo %q not found\n", repo)
				os.Exit(1)
			}

			entries, err := store.ReadSubject(subject)
			if err != nil {
				fmt.Printf("Error: reading subject %q: %v\n", subject, err)
				os.Exit(1)
			}

			found := false
			for i, e := range entries {
				if e.ID == id {
					if score > 0 {
						// Set AI weight (score)
						entries[i].Score = score
						entries[i].Starred = true
					} else {
						// Toggle starred flag
						entries[i].Starred = !entries[i].Starred
					}
					found = true
					break
				}
			}

			if !found {
				fmt.Printf("Error: entry #%d not found in subject %q\n", id, subject)
				os.Exit(1)
			}

			if err := store.WriteSubject(subject, entries); err != nil {
				fmt.Printf("Error: writing subject %q: %v\n", subject, err)
				os.Exit(1)
			}

			newScore := entries[findEntryIndex(entries, id)].Score
			if score > 0 {
				fmt.Printf("⭐ %s#%d AI weight set to %.2f\n", subject, id, newScore)
			} else {
				fmt.Printf("⭐ %s#%d starred (score: %.2f)\n", subject, id, newScore)
			}
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "Repository identifier (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject containing the entry (required)")
	cmd.Flags().IntVar(&id, "id", 0, "Entry ID to star (required)")
	cmd.Flags().Float64Var(&score, "score", 0.0, "AI weight score (0.0-1.0). Omit to toggle starred flag.")

	return cmd
}

func findEntryIndex(entries []memory.Entry, id int) int {
	for i, e := range entries {
		if e.ID == id {
			return i
		}
	}
	return 0
}
