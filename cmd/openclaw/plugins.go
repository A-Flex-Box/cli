package openclaw

import (
	"fmt"

	plug "github.com/A-Flex-Box/cli/internal/openclaw/plugin"
	"github.com/spf13/cobra"
)

func newPluginsCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plugins",
		Short:   "Manage OpenClaw plugins",
		Long:    `List, install, enable, and disable OpenClaw plugins.`,
		Example: "cli openclaw plugins list\n  cli openclaw plugins install github",
	}

	cmd.AddCommand(newPluginsListCmd(ctx))
	cmd.AddCommand(newPluginsInstallCmd(ctx))
	cmd.AddCommand(newPluginsUninstallCmd(ctx))
	cmd.AddCommand(newPluginsEnableCmd(ctx))
	cmd.AddCommand(newPluginsDisableCmd(ctx))

	return cmd
}

func newPluginsListCmd(ctx *Context) *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available plugins",
		Example: "cli openclaw plugins list\n  cli openclaw plugins list --category channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			plugins := ctx.PluginManager.List()
			if category != "" {
				plugins = ctx.PluginManager.ListByCategory(category)
			}

			fmt.Println("OpenClaw Plugins")
			fmt.Println("=================")
			fmt.Println()

			categories := []string{"channel", "tool", "skill"}
			for _, cat := range categories {
				catPlugins := filterPluginsByCategory(plugins, cat)
				if len(catPlugins) > 0 {
					fmt.Printf("%s Plugins:\n", titleCategory(cat))
					for _, p := range catPlugins {
						status := ""
						if p.Installed {
							if p.Enabled {
								status = "[installed, enabled]"
							} else {
								status = "[installed, disabled]"
							}
						}
						fmt.Printf("  %-12s %s %s\n", p.ID, p.Name, status)
						fmt.Printf("               %s\n", p.Description)
					}
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter by category: channel, tool, skill")

	return cmd
}

func newPluginsInstallCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install <plugin-id>",
		Short:   "Install a plugin",
		Args:    cobra.ExactArgs(1),
		Example: "cli openclaw plugins install github\n  cli openclaw plugins install whatsapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginID := args[0]

			p := ctx.PluginManager.GetByID(pluginID)
			if p == nil {
				return fmt.Errorf("plugin not found: %s", pluginID)
			}

			if p.Installed {
				fmt.Printf("Plugin %s is already installed.\n", pluginID)
				return nil
			}

			if err := ctx.PluginManager.Install(pluginID); err != nil {
				return err
			}

			fmt.Printf("Plugin %s installed successfully.\n", pluginID)
			return nil
		},
	}

	return cmd
}

func newPluginsUninstallCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "uninstall <plugin-id>",
		Short:   "Uninstall a plugin",
		Args:    cobra.ExactArgs(1),
		Example: "cli openclaw plugins uninstall github",
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginID := args[0]

			p := ctx.PluginManager.GetByID(pluginID)
			if p == nil {
				return fmt.Errorf("plugin not found: %s", pluginID)
			}

			if !p.Installed {
				fmt.Printf("Plugin %s is not installed.\n", pluginID)
				return nil
			}

			if err := ctx.PluginManager.Uninstall(pluginID); err != nil {
				return err
			}

			fmt.Printf("Plugin %s uninstalled.\n", pluginID)
			return nil
		},
	}

	return cmd
}

func newPluginsEnableCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <plugin-id>",
		Short:   "Enable a plugin",
		Args:    cobra.ExactArgs(1),
		Example: "cli openclaw plugins enable github",
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginID := args[0]

			p := ctx.PluginManager.GetByID(pluginID)
			if p == nil {
				return fmt.Errorf("plugin not found: %s", pluginID)
			}

			if !p.Installed {
				return fmt.Errorf("plugin %s is not installed", pluginID)
			}

			if err := ctx.PluginManager.Enable(pluginID); err != nil {
				return err
			}

			fmt.Printf("Plugin %s enabled.\n", pluginID)
			return nil
		},
	}

	return cmd
}

func newPluginsDisableCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disable <plugin-id>",
		Short:   "Disable a plugin",
		Args:    cobra.ExactArgs(1),
		Example: "cli openclaw plugins disable github",
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginID := args[0]

			p := ctx.PluginManager.GetByID(pluginID)
			if p == nil {
				return fmt.Errorf("plugin not found: %s", pluginID)
			}

			if err := ctx.PluginManager.Disable(pluginID); err != nil {
				return err
			}

			fmt.Printf("Plugin %s disabled.\n", pluginID)
			return nil
		},
	}

	return cmd
}

func filterPluginsByCategory(plugins []plug.Plugin, category string) []plug.Plugin {
	var result []plug.Plugin
	for _, p := range plugins {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}

func titleCategory(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
