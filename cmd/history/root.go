package history

import (
	"github.com/spf13/cobra"
)

// NewCmd returns the history parent command with add subcommand.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Manage project history",
	}
	cmd.AddCommand(newAddCmd())
	return cmd
}
