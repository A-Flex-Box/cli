package cmd

import (
	"github.com/A-Flex-Box/cli/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment health (tools and services)",
	Long:  `Detect installed tools (go, git, make, gcc, cpp, py, conda) and services (docker, containerd, k8s, etcd, mysql, pg, es) with versions and default port status.`,
	Run: func(cmd *cobra.Command, args []string) {
		doctor.Run()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
