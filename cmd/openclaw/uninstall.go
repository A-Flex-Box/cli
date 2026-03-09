package openclaw

import (
	"context"
	"fmt"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
	"github.com/spf13/cobra"
)

func newUninstallCmd(ctx *Context) *cobra.Command {
	var opts installer.UninstallOptions

	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall OpenClaw",
		Long:    `Remove OpenClaw from your system.`,
		Example: "cli openclaw uninstall\n  cli openclaw uninstall --purge",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isGuiMode(cmd) {
				return runUninstallGUI(ctx, &opts)
			}
			return runUninstallCLI(cmd, ctx, &opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Purge, "purge", false, "Remove all configuration and data")
	cmd.Flags().BoolVar(&opts.NoGui, "no-gui", false, "Force command-line mode")

	return cmd
}

func runUninstallCLI(cmd *cobra.Command, ctx *Context, opts *installer.UninstallOptions) error {
	progress := func(stage string, current, total int, message string) {
		if total > 0 {
			percent := (current * 100) / total
			fmt.Printf("[%s] %d/%d - %s (%d%%)\n", stage, current, total, message, percent)
		} else {
			fmt.Printf("[%s] %s\n", stage, message)
		}
	}

	fmt.Println("OpenClaw Uninstaller")
	fmt.Println("====================")

	if opts.Purge {
		fmt.Println("WARNING: --purge will remove ALL configuration and data!")
	}

	inst := installer.NewInstaller(installer.MethodNative, ctx.SysInfo)
	if !inst.IsInstalled() {
		fmt.Println("OpenClaw is not installed.")
		return nil
	}

	fmt.Print("Are you sure you want to uninstall OpenClaw? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	fmt.Println("Starting uninstallation...")
	err := inst.Uninstall(context.Background(), opts, progress)
	if err != nil {
		return fmt.Errorf("uninstallation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("OpenClaw has been uninstalled.")
	return nil
}

func runUninstallGUI(ctx *Context, opts *installer.UninstallOptions) error {
	return runUninstallCLI(nil, ctx, opts)
}
