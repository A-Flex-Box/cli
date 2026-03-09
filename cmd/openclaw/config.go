package openclaw

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/A-Flex-Box/cli/internal/openclaw/config"
	"github.com/spf13/cobra"
)

func newConfigCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Manage OpenClaw configuration",
		Long:    `View and modify OpenClaw configuration settings.`,
		Example: "cli openclaw config show\n  cli openclaw config set agent.model gpt-4",
	}

	cmd.AddCommand(newConfigShowCmd(ctx))
	cmd.AddCommand(newConfigSetCmd(ctx))
	cmd.AddCommand(newConfigInitCmd(ctx))

	return cmd
}

func newConfigShowCmd(ctx *Context) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Show current configuration",
		Example: "cli openclaw config show\n  cli openclaw config show --format json",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := getConfigPath()
			data, err := os.ReadFile(configPath)
			if err != nil {
				fmt.Println("No configuration file found. Using defaults:")
				printConfig(ctx.Config, format)
				return nil
			}

			var cfg config.OpenClawConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}

			printConfig(&cfg, format)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "yaml", "Output format: yaml, json")

	return cmd
}

func newConfigSetCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set <key> <value>",
		Short:   "Set a configuration value",
		Example: "cli openclaw config set agent.model claude-opus-4\n  cli openclaw config set gateway.port 8080",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			configPath := getConfigPath()
			cfg := loadConfig(configPath)

			if err := setConfigValue(cfg, key, value); err != nil {
				return err
			}

			return saveConfig(configPath, cfg)
		},
	}

	return cmd
}

func newConfigInitCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize configuration with defaults",
		Example: "cli openclaw config init",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := getConfigPath()

			if _, err := os.Stat(configPath); err == nil {
				fmt.Print("Configuration already exists. Overwrite? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					return nil
				}
			}

			cfg := config.DefaultConfig()
			if err := saveConfig(configPath, cfg); err != nil {
				return err
			}

			fmt.Printf("Configuration initialized at %s\n", configPath)
			return nil
		},
	}

	return cmd
}

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".openclaw", "config.json")
}

func loadConfig(path string) *config.OpenClawConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.DefaultConfig()
	}

	var cfg config.OpenClawConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.DefaultConfig()
	}

	return &cfg
}

func saveConfig(path string, cfg *config.OpenClawConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func printConfig(cfg *config.OpenClawConfig, format string) {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
	default:
		fmt.Printf("Agent:\n")
		fmt.Printf("  Model: %s\n", cfg.Agent.Model)
		fmt.Printf("  Workspace: %s\n", cfg.Agent.Workspace)
		fmt.Printf("  Thinking: %s\n", cfg.Agent.Thinking)
		fmt.Printf("Gateway:\n")
		fmt.Printf("  Port: %d\n", cfg.Gateway.Port)
		fmt.Printf("  Bind: %s\n", cfg.Gateway.Bind)
		fmt.Printf("  Auth Mode: %s\n", cfg.Gateway.AuthMode)
		fmt.Printf("Tools:\n")
		fmt.Printf("  Browser: %v\n", cfg.Tools.BrowserEnabled)
		fmt.Printf("  Voice: %v\n", cfg.Tools.VoiceEnabled)
		fmt.Printf("Plugins:\n")
		fmt.Printf("  Enabled: %v\n", cfg.Plugins.Enabled)
	}
}

func setConfigValue(cfg *config.OpenClawConfig, key, value string) error {
	switch key {
	case "agent.model":
		cfg.Agent.Model = value
	case "agent.workspace":
		cfg.Agent.Workspace = value
	case "agent.thinking":
		cfg.Agent.Thinking = value
	case "gateway.port":
		fmt.Sscanf(value, "%d", &cfg.Gateway.Port)
	case "gateway.bind":
		cfg.Gateway.Bind = value
	case "gateway.authMode":
		cfg.Gateway.AuthMode = value
	case "gateway.token":
		cfg.Gateway.Token = value
	case "tools.browserEnabled":
		cfg.Tools.BrowserEnabled = value == "true"
	case "tools.voiceEnabled":
		cfg.Tools.VoiceEnabled = value == "true"
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}
