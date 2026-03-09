package openclaw

import (
	"github.com/A-Flex-Box/cli/internal/openclaw/config"
	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
	"github.com/A-Flex-Box/cli/internal/openclaw/plugin"
	"github.com/A-Flex-Box/cli/internal/openclaw/service"
	"github.com/spf13/cobra"
)

type Context struct {
	SysInfo       *installer.SystemInfo
	Config        *config.OpenClawConfig
	PluginManager *plugin.Manager
	ServiceMgr    *service.Manager
}

func NewCmd() *cobra.Command {
	ctx := &Context{
		SysInfo:       installer.DetectSystem(),
		Config:        config.DefaultConfig(),
		PluginManager: plugin.NewManager(),
	}

	cmd := &cobra.Command{
		Use:     "openclaw",
		Short:   "OpenClaw deployment and management",
		Long:    `Install, configure, and manage OpenClaw - the AI agent framework.`,
		Example: "cli openclaw install\n  cli openclaw config\n  cli openclaw plugins list",
	}

	cmd.PersistentFlags().Bool("gui", false, "Launch GUI mode (auto-detects display)")
	cmd.PersistentFlags().Bool("no-gui", false, "Force command-line mode")

	cmd.AddCommand(newInstallCmd(ctx))
	cmd.AddCommand(newUninstallCmd(ctx))
	cmd.AddCommand(newConfigCmd(ctx))
	cmd.AddCommand(newPluginsCmd(ctx))
	cmd.AddCommand(newServiceCmd(ctx))
	cmd.AddCommand(newUpdateCmd(ctx))
	cmd.AddCommand(newDoctorCmd(ctx))
	cmd.AddCommand(newVersionCmd(ctx))

	return cmd
}

func isGuiMode(cmd *cobra.Command) bool {
	guiFlag, _ := cmd.Flags().GetBool("gui")
	noGuiFlag, _ := cmd.Flags().GetBool("no-gui")

	if noGuiFlag {
		return false
	}
	if guiFlag {
		return true
	}

	sysInfo := installer.DetectSystem()
	return sysInfo.HasDisplay()
}
