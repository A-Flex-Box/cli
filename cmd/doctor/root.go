package doctor

import (
	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/spf13/cobra"
)

// NewCmd returns the doctor command (parent) with subcommands check, port, watch.
func NewCmd(cfg *config.Root) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "doctor",
		Short:   "Network Diagnostic & Instrumentation Suite",
		Long:    `Diagnose connectivity, inspect ports, and monitor network traffic. Foundation for Wormhole traffic tracing.`,
		Example: "cli doctor check\n  cli doctor port 8080\n  cli doctor watch",
		RunE: func(c *cobra.Command, args []string) error {
			return runCheck(cfg)
		},
	}
	cmd.AddCommand(newCheckCmd(cfg))
	cmd.AddCommand(newPortCmd())
	cmd.AddCommand(newWatchCmd())
	return cmd
}
