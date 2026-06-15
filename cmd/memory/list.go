package memory

import (
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/memory"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all subjects in a repo",
		Long:  `Display the INDEX table for a memory repo, showing all subjects and their metadata.`,
		Example: `  cli memory list --repo dot`,
		Run: func(cmd *cobra.Command, args []string) {
			if repo == "" {
				fmt.Println("Error: --repo is required")
				os.Exit(1)
			}

			store := memory.NewStore(repo)

			if !store.RepoExists() {
				fmt.Printf("No memory repo %q found\n", repo)
				return
			}

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
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "Repository identifier (required)")

	return cmd
}
