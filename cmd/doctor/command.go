package doctor

import (
	"github.com/A-Flex-Box/cli/internal/doctor"
	"github.com/spf13/cobra"
)

// NewCmd returns the doctor command.
func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "doctor",
		Short:   "Check environment health (tools and services)",
		Long:    `Detect installed tools (go, git, make, gcc, cpp, py, conda) and services (docker, containerd, k8s, etcd, mysql, pg, es) with versions and default port status.`,
		Example: "cli doctor",
		Run: func(cmd *cobra.Command, args []string) {
			doctor.Run()
		},
	}
}
