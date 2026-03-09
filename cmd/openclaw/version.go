package openclaw

import (
	"fmt"
	"os/exec"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
	"github.com/spf13/cobra"
)

func newVersionCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Show OpenClaw version",
		Example: "cli openclaw version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(ctx)
		},
	}

	return cmd
}

func runVersion(ctx *Context) error {
	fmt.Println("OpenClaw Version")
	fmt.Println("================")

	inst := installer.NewInstaller(installer.MethodNative, ctx.SysInfo)
	if inst.IsInstalled() {
		version, err := inst.GetVersion()
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}
		fmt.Printf("OpenClaw: %s\n", version)

		location, err := exec.LookPath("openclaw")
		if err == nil {
			fmt.Printf("Location: %s\n", location)
		}
	} else {
		fmt.Println("OpenClaw is not installed.")
		fmt.Println("Run 'cli openclaw install' to install.")
	}

	return nil
}
