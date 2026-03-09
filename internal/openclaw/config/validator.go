package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func ValidateConfig(cfg *OpenClawConfig) ValidationErrors {
	var errs ValidationErrors

	errs = append(errs, validateAgentConfig(&cfg.Agent)...)
	errs = append(errs, validateGatewayConfig(&cfg.Gateway)...)
	errs = append(errs, validateToolsConfig(&cfg.Tools)...)
	errs = append(errs, validatePluginConfig(&cfg.Plugins)...)

	return errs
}

func validateAgentConfig(cfg *AgentConfig) ValidationErrors {
	var errs ValidationErrors

	if cfg.Model == "" {
		errs = append(errs, ValidationError{
			Field:   "agent.model",
			Message: "model cannot be empty",
		})
	}

	validModels := []string{
		"anthropic/claude-opus-4-6",
		"anthropic/claude-sonnet-4-6",
		"anthropic/claude-haiku-4-6",
		"openai/gpt-4o",
		"openai/gpt-4o-mini",
		"google/gemini-2.0-flash",
	}
	modelValid := false
	for _, m := range validModels {
		if cfg.Model == m || strings.HasPrefix(cfg.Model, m) {
			modelValid = true
			break
		}
	}
	if !modelValid && cfg.Model != "" {
		errs = append(errs, ValidationError{
			Field:   "agent.model",
			Message: fmt.Sprintf("unknown model: %s", cfg.Model),
		})
	}

	if cfg.Workspace == "" {
		errs = append(errs, ValidationError{
			Field:   "agent.workspace",
			Message: "workspace cannot be empty",
		})
	} else {
		expanded := expandPath(cfg.Workspace)
		if !filepath.IsAbs(expanded) {
			errs = append(errs, ValidationError{
				Field:   "agent.workspace",
				Message: "workspace must be an absolute path",
			})
		}
	}

	validThinking := map[string]bool{"low": true, "medium": true, "high": true}
	if !validThinking[cfg.Thinking] {
		errs = append(errs, ValidationError{
			Field:   "agent.thinking",
			Message: "thinking must be one of: low, medium, high",
		})
	}

	return errs
}

func validateGatewayConfig(cfg *GatewayConfig) ValidationErrors {
	var errs ValidationErrors

	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, ValidationError{
			Field:   "gateway.port",
			Message: "port must be between 1 and 65535",
		})
	}

	if cfg.Port < 1024 {
		errs = append(errs, ValidationError{
			Field:   "gateway.port",
			Message: "port below 1024 requires root privileges",
		})
	}

	if cfg.Bind != "loopback" && cfg.Bind != "all" && cfg.Bind != "localhost" && cfg.Bind != "0.0.0.0" {
		if net.ParseIP(cfg.Bind) == nil {
			errs = append(errs, ValidationError{
				Field:   "gateway.bind",
				Message: "bind must be 'loopback', 'all', or a valid IP address",
			})
		}
	}

	if cfg.AuthMode != "token" && cfg.AuthMode != "none" {
		errs = append(errs, ValidationError{
			Field:   "gateway.authMode",
			Message: "authMode must be 'token' or 'none'",
		})
	}

	if cfg.AuthMode == "token" && cfg.Token == "" {
		errs = append(errs, ValidationError{
			Field:   "gateway.token",
			Message: "token is required when authMode is 'token'",
		})
	}

	return errs
}

func validateToolsConfig(cfg *ToolsConfig) ValidationErrors {
	var errs ValidationErrors

	if cfg.VoiceEnabled && !cfg.BrowserEnabled {
		errs = append(errs, ValidationError{
			Field:   "tools.voiceEnabled",
			Message: "voice requires browser to be enabled",
		})
	}

	return errs
}

func validatePluginConfig(cfg *PluginConfig) ValidationErrors {
	var errs ValidationErrors

	seen := make(map[string]bool)
	for _, plugin := range cfg.Enabled {
		if seen[plugin] {
			errs = append(errs, ValidationError{
				Field:   "plugins.enabled",
				Message: fmt.Sprintf("duplicate plugin: %s", plugin),
			})
		}
		seen[plugin] = true
	}

	validPlugins := map[string]bool{
		"github": true, "slack": true, "discord": true, "telegram": true,
		"whatsapp": true, "notion": true, "obsidian": true, "spotify": true,
		"weather": true, "browser": true, "voice": true, "canvas": true,
		"calendar": true, "email": true, "signal": true,
	}

	for _, plugin := range cfg.Enabled {
		if !validPlugins[plugin] {
			errs = append(errs, ValidationError{
				Field:   "plugins.enabled",
				Message: fmt.Sprintf("unknown plugin: %s", plugin),
			})
		}
	}

	return errs
}

func ValidateConfigPath(path string) error {
	expanded := expandPath(path)

	if !filepath.IsAbs(expanded) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	dir := filepath.Dir(expanded)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func (errs ValidationErrors) HasErrors() bool {
	return len(errs) > 0
}

func (errs ValidationErrors) GetFieldErrors(field string) []ValidationError {
	var result []ValidationError
	for _, err := range errs {
		if err.Field == field {
			result = append(result, err)
		}
	}
	return result
}
