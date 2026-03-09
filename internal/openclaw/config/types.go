package config

type OpenClawConfig struct {
	Agent   AgentConfig   `json:"agent"`
	Gateway GatewayConfig `json:"gateway"`
	Tools   ToolsConfig   `json:"tools"`
	Plugins PluginConfig  `json:"plugins"`
}

type AgentConfig struct {
	Model     string `json:"model"`
	Workspace string `json:"workspace"`
	Thinking  string `json:"thinking"`
}

type GatewayConfig struct {
	Port     int    `json:"port"`
	Bind     string `json:"bind"`
	AuthMode string `json:"authMode"`
	Token    string `json:"token"`
}

type ToolsConfig struct {
	BrowserEnabled bool `json:"browserEnabled"`
	VoiceEnabled   bool `json:"voiceEnabled"`
}

type PluginConfig struct {
	Enabled []string          `json:"enabled"`
	ApiKey  map[string]string `json:"apiKey"`
}

func DefaultConfig() *OpenClawConfig {
	return &OpenClawConfig{
		Agent: AgentConfig{
			Model:     "anthropic/claude-opus-4-6",
			Workspace: "~/.openclaw/workspace",
			Thinking:  "medium",
		},
		Gateway: GatewayConfig{
			Port:     18789,
			Bind:     "loopback",
			AuthMode: "token",
		},
		Tools: ToolsConfig{
			BrowserEnabled: true,
			VoiceEnabled:   false,
		},
		Plugins: PluginConfig{
			Enabled: []string{},
			ApiKey:  make(map[string]string),
		},
	}
}
