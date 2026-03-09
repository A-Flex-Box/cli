package openclaw

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
	"github.com/spf13/cobra"
)

func newUpdateCmd(ctx *Context) *cobra.Command {
	var version string
	var force bool

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update OpenClaw",
		Long:    `Update OpenClaw to the latest version.`,
		Example: "cli openclaw update\n  cli openclaw update --version beta",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isGuiMode(cmd) {
				return runUpdateGUI(ctx, version, force)
			}
			return runUpdateCLI(cmd, ctx, version, force)
		},
	}

	cmd.Flags().StringVar(&version, "version", "stable", "Target version: stable, beta, dev")
	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already up to date")

	return cmd
}

func runUpdateCLI(cmd *cobra.Command, ctx *Context, targetVersion string, force bool) error {
	progress := func(stage string, current, total int, message string) {
		if total > 0 {
			percent := (current * 100) / total
			fmt.Printf("[%s] %d/%d - %s (%d%%)\n", stage, current, total, message, percent)
		} else {
			fmt.Printf("[%s] %s\n", stage, message)
		}
	}

	fmt.Println("OpenClaw Updater")
	fmt.Println("================")

	inst := installer.NewInstaller(installer.MethodNative, ctx.SysInfo)
	if !inst.IsInstalled() {
		return fmt.Errorf("OpenClaw is not installed. Run 'cli openclaw install' first")
	}

	currentVersion, _ := inst.GetVersion()
	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Target version: %s\n", targetVersion)
	fmt.Println()

	if ctx.SysInfo.NpmVersion == "" {
		return fmt.Errorf("npm is required for updates")
	}

	fmt.Println("Updating OpenClaw...")

	var updateCmd *exec.Cmd
	switch targetVersion {
	case "beta":
		updateCmd = exec.CommandContext(context.Background(), "npm", "update", "-g", "openclaw@beta")
	case "dev":
		updateCmd = exec.CommandContext(context.Background(), "npm", "update", "-g", "openclaw@next")
	default:
		updateCmd = exec.CommandContext(context.Background(), "npm", "update", "-g", "openclaw@latest")
	}

	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr

	progress("update", 1, 3, "Downloading update...")
	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	progress("update", 2, 3, "Verifying installation...")
	newVersion, err := inst.GetVersion()
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	progress("update", 3, 3, fmt.Sprintf("Updated to %s", newVersion))

	fmt.Println()
	fmt.Printf("OpenClaw updated to version %s\n", newVersion)
	return nil
}

func runUpdateGUI(ctx *Context, version string, force bool) error {
	return runUpdateCLI(nil, ctx, version, force)
}
