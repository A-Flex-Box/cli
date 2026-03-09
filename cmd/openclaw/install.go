package openclaw

import (
	"context"
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
	"github.com/spf13/cobra"
)

func newInstallCmd(ctx *Context) *cobra.Command {
	var opts installer.InstallOptions
	var components []string

	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install OpenClaw",
		Long:    `Install OpenClaw using the specified method (native, docker, or source).`,
		Example: "cli openclaw install\n  cli openclaw install --method docker\n  cli openclaw install --components whatsapp,telegram",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Components = components

			if isGuiMode(cmd) {
				return runInstallGUI(ctx, &opts)
			}
			return runInstallCLI(cmd, ctx, &opts)
		},
	}

	cmd.Flags().StringVar((*string)(&opts.Method), "method", "native", "Installation method: native, docker, source")
	cmd.Flags().StringVar((*string)(&opts.Version), "version", "stable", "Version to install: stable, beta, dev")
	cmd.Flags().StringVar(&opts.InstallPath, "path", "", "Custom installation path")
	cmd.Flags().BoolVar(&opts.NoDaemon, "no-daemon", false, "Skip daemon/service installation")
	cmd.Flags().BoolVar(&opts.NoGui, "no-gui", false, "Skip GUI components")
	cmd.Flags().StringSliceVar(&components, "components", []string{}, "Additional components to install")

	return cmd
}

func runInstallCLI(cmd *cobra.Command, ctx *Context, opts *installer.InstallOptions) error {
	progress := func(stage string, current, total int, message string) {
		if total > 0 {
			percent := (current * 100) / total
			fmt.Printf("[%s] %d/%d - %s (%d%%)\n", stage, current, total, message, percent)
		} else {
			fmt.Printf("[%s] %s\n", stage, message)
		}
	}

	fmt.Println("OpenClaw Installer")
	fmt.Println("==================")
	fmt.Printf("Method: %s\n", opts.Method)
	fmt.Printf("Version: %s\n", opts.Version)
	fmt.Println()

	depErrors := ctx.SysInfo.CheckDependencies(opts.Method)
	if len(depErrors) > 0 {
		fmt.Println("Missing dependencies:")
		for _, depErr := range depErrors {
			status := "optional"
			if depErr.Required {
				status = "required"
			}
			fmt.Printf("  - %s (%s): %s\n", depErr.Name, status, depErr.Description)
			fmt.Printf("    Fix: %s\n", depErr.HowToFix)
		}
		for _, depErr := range depErrors {
			if depErr.Required {
				return fmt.Errorf("missing required dependency: %s", depErr.Name)
			}
		}
	}

	inst := installer.NewInstaller(opts.Method, ctx.SysInfo)
	if inst.IsInstalled() {
		version, _ := inst.GetVersion()
		fmt.Printf("OpenClaw is already installed: %s\n", version)
		fmt.Print("Reinstall? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return nil
		}
	}

	fmt.Println("Starting installation...")
	err := inst.Install(context.Background(), opts, progress)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("Installation complete!")
	return nil
}

func runInstallGUI(ctx *Context, opts *installer.InstallOptions) error {
	fmt.Fprintln(os.Stderr, "GUI mode not available. Running in CLI mode.")
	return runInstallCLI(nil, ctx, opts)
}
