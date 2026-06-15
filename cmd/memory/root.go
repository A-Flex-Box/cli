package memory

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Persistent session memory",
		Long:  `Save and load structured memory (findings, hypotheses, ideas, etc.) across sessions.`,
		Example: `  cli memory save --repo dot --subject hypr --type findings "Super key triggers on release"
  cli memory load --repo dot
  cli memory list --repo dot
  cli memory search --repo dot --keyword "keybinding"`,
	}
	cmd.AddCommand(newSaveCmd())
	cmd.AddCommand(newLoadCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newPruneCmd())
	cmd.AddCommand(newStarCmd())
	return cmd
}
