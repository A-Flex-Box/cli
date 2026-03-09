package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func GenerateConfig(path string, cfg *OpenClawConfig) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func LoadConfig(path string) (*OpenClawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg OpenClawConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".openclaw", "config.json"), nil
}

func EnsureConfigExists() (*OpenClawConfig, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := GenerateConfig(path, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	return LoadConfig(path)
}

func GenerateAgentConfig(workspace string, model string, thinking string) AgentConfig {
	return AgentConfig{
		Model:     model,
		Workspace: workspace,
		Thinking:  thinking,
	}
}

func GenerateGatewayConfig(port int, bind string, authMode string, token string) GatewayConfig {
	if port == 0 {
		port = 18789
	}
	if bind == "" {
		bind = "loopback"
	}
	if authMode == "" {
		authMode = "token"
	}
	return GatewayConfig{
		Port:     port,
		Bind:     bind,
		AuthMode: authMode,
		Token:    token,
	}
}

func GeneratePluginConfig(enabled []string, apiKeys map[string]string) PluginConfig {
	if enabled == nil {
		enabled = []string{}
	}
	if apiKeys == nil {
		apiKeys = make(map[string]string)
	}
	return PluginConfig{
		Enabled: enabled,
		ApiKey:  apiKeys,
	}
}
