package monitor

import (
	"github.com/spf13/cobra"
)

// NewCmd returns the monitor command with subcommands.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "monitor",
		Short:   "Network traffic monitoring TUI",
		Long:    `Real-time network monitoring with packet capture, route tracking, and connection analysis.`,
		Example: "cli monitor\n  cli monitor --interface eth0\n  cli monitor --no-capture",
		RunE:    runMonitor,
	}

	// Flags for the main TUI command
	cmd.Flags().StringP("interface", "i", "", "Network interface to capture (empty = auto-detect)")
	cmd.Flags().Bool("no-capture", false, "Disable packet capture")
	cmd.Flags().Bool("no-routes", false, "Disable route monitoring")
	cmd.Flags().Bool("no-conntrack", false, "Disable connection tracking")
	cmd.Flags().StringP("filter", "f", "", "BPF filter for packet capture")
	cmd.Flags().IntP("refresh", "r", 1, "Refresh interval in seconds")

	return cmd
}

func runMonitor(cmd *cobra.Command, args []string) error {
	iface, _ := cmd.Flags().GetString("interface")
	noCapture, _ := cmd.Flags().GetBool("no-capture")
	noRoutes, _ := cmd.Flags().GetBool("no-routes")
	noConntrack, _ := cmd.Flags().GetBool("no-conntrack")
	filter, _ := cmd.Flags().GetString("filter")
	refresh, _ := cmd.Flags().GetInt("refresh")

	opts := &Options{
		Interface:    iface,
		EnableCapture: !noCapture,
		EnableRoutes: !noRoutes,
		EnableConntrack: !noConntrack,
		BPFFilter:    filter,
		RefreshRate:  refresh,
	}

	return RunMonitorTUI(opts)
}

// Options holds configuration for the monitor.
type Options struct {
	Interface       string
	EnableCapture   bool
	EnableRoutes    bool
	EnableConntrack bool
	BPFFilter       string
	RefreshRate     int
}