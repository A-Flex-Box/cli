package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agent.Model == "" {
		t.Error("expected default model to be set")
	}
	if cfg.Agent.Workspace == "" {
		t.Error("expected default workspace to be set")
	}
	if cfg.Gateway.Port == 0 {
		t.Error("expected default port to be set")
	}
	if cfg.Gateway.Port != 18789 {
		t.Errorf("expected default port 18789, got %d", cfg.Gateway.Port)
	}
	if cfg.Agent.Thinking != "medium" {
		t.Errorf("expected default thinking 'medium', got %s", cfg.Agent.Thinking)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *OpenClawConfig
		wantErrs int
	}{
		{
			name: "valid config",
			config: func() *OpenClawConfig {
				cfg := DefaultConfig()
				cfg.Gateway.Token = "test-token"
				return cfg
			}(),
			wantErrs: 0,
		},
		{
			name: "empty model",
			config: &OpenClawConfig{
				Agent: AgentConfig{
					Model:     "",
					Workspace: "/tmp/workspace",
					Thinking:  "medium",
				},
				Gateway: GatewayConfig{
					Port:     18789,
					Bind:     "loopback",
					AuthMode: "token",
					Token:    "test-token",
				},
			},
			wantErrs: 1,
		},
		{
			name: "invalid port",
			config: &OpenClawConfig{
				Agent: AgentConfig{
					Model:     "anthropic/claude-opus-4-6",
					Workspace: "/tmp/workspace",
					Thinking:  "medium",
				},
				Gateway: GatewayConfig{
					Port:     99999,
					Bind:     "loopback",
					AuthMode: "token",
					Token:    "test-token",
				},
			},
			wantErrs: 1,
		},
		{
			name: "invalid thinking",
			config: &OpenClawConfig{
				Agent: AgentConfig{
					Model:     "anthropic/claude-opus-4-6",
					Workspace: "/tmp/workspace",
					Thinking:  "invalid",
				},
				Gateway: GatewayConfig{
					Port:     18789,
					Bind:     "loopback",
					AuthMode: "token",
					Token:    "test-token",
				},
			},
			wantErrs: 1,
		},
		{
			name: "voice without browser",
			config: &OpenClawConfig{
				Agent: AgentConfig{
					Model:     "anthropic/claude-opus-4-6",
					Workspace: "/tmp/workspace",
					Thinking:  "medium",
				},
				Gateway: GatewayConfig{
					Port:     18789,
					Bind:     "loopback",
					AuthMode: "token",
					Token:    "test-token",
				},
				Tools: ToolsConfig{
					BrowserEnabled: false,
					VoiceEnabled:   true,
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateConfig(tt.config)
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateConfig() got %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestGenerateAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.Agent.Workspace = "/custom/workspace"
	cfg.Gateway.Port = 8080

	if err := GenerateConfig(configPath, cfg); err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Agent.Workspace != "/custom/workspace" {
		t.Errorf("expected workspace '/custom/workspace', got %s", loaded.Agent.Workspace)
	}
	if loaded.Gateway.Port != 8080 {
		t.Errorf("expected port 8080, got %d", loaded.Gateway.Port)
	}
}

func TestValidationErrors(t *testing.T) {
	errs := ValidationErrors{
		{Field: "test", Message: "error 1"},
		{Field: "test", Message: "error 2"},
	}

	if errs.Error() == "" {
		t.Error("ValidationErrors.Error() should not be empty")
	}

	if !errs.HasErrors() {
		t.Error("ValidationErrors.HasErrors() should be true")
	}

	fieldErrs := errs.GetFieldErrors("test")
	if len(fieldErrs) != 2 {
		t.Errorf("expected 2 field errors, got %d", len(fieldErrs))
	}
}
