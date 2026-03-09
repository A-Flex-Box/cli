package openclaw

import (
	"fmt"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
	"github.com/spf13/cobra"
)

func newDoctorCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "doctor",
		Short:   "Diagnose OpenClaw installation",
		Long:    `Run diagnostic checks on your OpenClaw installation.`,
		Example: "cli openclaw doctor",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(ctx)
		},
	}

	return cmd
}

func runDoctor(ctx *Context) error {
	fmt.Println("OpenClaw Diagnostic")
	fmt.Println("===================")
	fmt.Println()

	fmt.Println("System Information:")
	fmt.Printf("  OS: %s\n", ctx.SysInfo.OS)
	fmt.Printf("  Arch: %s\n", ctx.SysInfo.Arch)
	if ctx.SysInfo.Distro != "" {
		fmt.Printf("  Distro: %s %s\n", ctx.SysInfo.Distro, ctx.SysInfo.Version)
	}
	fmt.Println()

	fmt.Println("Dependencies:")
	checkDependency("Node.js", ctx.SysInfo.NodeVersion)
	checkDependency("npm", ctx.SysInfo.NpmVersion)
	checkDependency("Docker", ctx.SysInfo.DockerVer)
	checkDependency("Git", ctx.SysInfo.GitVersion)
	fmt.Println()

	fmt.Println("Installation Status:")
	inst := installer.NewInstaller(installer.MethodNative, ctx.SysInfo)
	if inst.IsInstalled() {
		version, err := inst.GetVersion()
		if err != nil {
			fmt.Println("  [ERROR] OpenClaw installed but version check failed")
		} else {
			fmt.Printf("  [OK] OpenClaw %s installed\n", version)
		}
	} else {
		fmt.Println("  [INFO] OpenClaw not installed")
	}
	fmt.Println()

	fmt.Println("Service Status:")
	svcMgr := getServiceManager(ctx)
	info := svcMgr.Status()
	fmt.Printf("  Status: %s\n", info.Status)
	fmt.Printf("  Port: %d\n", info.Port)
	fmt.Println()

	fmt.Println("Network:")
	ports := []int{18789, 18790}
	inUse := ctx.SysInfo.DetectPortsInUse(ports)
	if len(inUse) > 0 {
		fmt.Printf("  [WARN] Ports in use: %v\n", inUse)
	} else {
		fmt.Println("  [OK] Required ports available")
	}
	fmt.Println()

	if ctx.SysInfo.Tailscale {
		fmt.Println("  [OK] Tailscale detected")
	}

	return nil
}

func checkDependency(name, version string) {
	if version != "" {
		fmt.Printf("  [OK] %s: %s\n", name, version)
	} else {
		fmt.Printf("  [MISSING] %s: not installed\n", name)
	}
}
