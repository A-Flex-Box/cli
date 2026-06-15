package memory

import (
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/memory"
	"github.com/spf13/cobra"
)

func newPruneCmd() *cobra.Command {
	var repo, subject string
	var max int

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune excess entries from a subject",
		Long: `Keep only the top N scored entries in a subject and move the rest to <subject>-trash.md.`,
		Example: `  cli memory prune --repo dot --subject hypr --max 50`,
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

			pruned, err := store.Prune(subject, max)
			if err != nil {
				fmt.Printf("Error: pruning subject %q: %v\n", subject, err)
				os.Exit(1)
			}

			fmt.Printf("✅ Pruned %d entries to %s-trash.md\n", pruned, subject)
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "Repository identifier (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject to prune (required)")
	cmd.Flags().IntVar(&max, "max", 50, "Maximum entries to keep")

	return cmd
}
