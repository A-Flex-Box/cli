package plugin

type BundledPlugin struct {
	ID          string
	Name        string
	Description string
	Category    string
	Version     string
	Author      string
	Repository  string
	ConfigKeys  []ConfigKey
}

type ConfigKey struct {
	Key         string
	Name        string
	Description string
	Required    bool
	Secret      bool
}

var BundledDefinitions = []BundledPlugin{
	{
		ID:          "github",
		Name:        "GitHub",
		Description: "GitHub repository and issue management",
		Category:    "skill",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/github",
		ConfigKeys: []ConfigKey{
			{Key: "token", Name: "Personal Access Token", Description: "GitHub PAT with repo scope", Required: true, Secret: true},
		},
	},
	{
		ID:          "slack",
		Name:        "Slack",
		Description: "Slack workspace integration",
		Category:    "channel",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/slack",
		ConfigKeys: []ConfigKey{
			{Key: "bot_token", Name: "Bot Token", Description: "Slack Bot User OAuth Token", Required: true, Secret: true},
			{Key: "app_token", Name: "App Token", Description: "Slack App-Level Token", Required: false, Secret: true},
		},
	},
	{
		ID:          "discord",
		Name:        "Discord",
		Description: "Discord bot integration",
		Category:    "channel",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/discord",
		ConfigKeys: []ConfigKey{
			{Key: "bot_token", Name: "Bot Token", Description: "Discord Bot Token", Required: true, Secret: true},
			{Key: "client_id", Name: "Client ID", Description: "Discord Application Client ID", Required: true, Secret: false},
		},
	},
	{
		ID:          "telegram",
		Name:        "Telegram",
		Description: "Telegram bot integration",
		Category:    "channel",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/telegram",
		ConfigKeys: []ConfigKey{
			{Key: "bot_token", Name: "Bot Token", Description: "Telegram Bot Token", Required: true, Secret: true},
		},
	},
	{
		ID:          "whatsapp",
		Name:        "WhatsApp",
		Description: "WhatsApp messaging integration",
		Category:    "channel",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/whatsapp",
		ConfigKeys: []ConfigKey{
			{Key: "session_path", Name: "Session Path", Description: "Path to WhatsApp session file", Required: true, Secret: false},
		},
	},
	{
		ID:          "notion",
		Name:        "Notion",
		Description: "Notion workspace integration",
		Category:    "skill",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/notion",
		ConfigKeys: []ConfigKey{
			{Key: "api_key", Name: "API Key", Description: "Notion Integration Secret", Required: true, Secret: true},
		},
	},
	{
		ID:          "obsidian",
		Name:        "Obsidian",
		Description: "Obsidian notes integration",
		Category:    "skill",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/obsidian",
		ConfigKeys: []ConfigKey{
			{Key: "vault_path", Name: "Vault Path", Description: "Path to Obsidian vault", Required: true, Secret: false},
		},
	},
	{
		ID:          "spotify",
		Name:        "Spotify",
		Description: "Spotify music control",
		Category:    "skill",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/spotify",
		ConfigKeys: []ConfigKey{
			{Key: "client_id", Name: "Client ID", Description: "Spotify App Client ID", Required: true, Secret: false},
			{Key: "client_secret", Name: "Client Secret", Description: "Spotify App Client Secret", Required: true, Secret: true},
			{Key: "refresh_token", Name: "Refresh Token", Description: "OAuth Refresh Token", Required: true, Secret: true},
		},
	},
	{
		ID:          "weather",
		Name:        "Weather",
		Description: "Weather information lookup",
		Category:    "skill",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/weather",
		ConfigKeys: []ConfigKey{
			{Key: "api_key", Name: "API Key", Description: "Weather API Key (OpenWeatherMap)", Required: false, Secret: true},
		},
	},
	{
		ID:          "browser",
		Name:        "Browser",
		Description: "Web browser automation and control",
		Category:    "tool",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/browser",
		ConfigKeys:  []ConfigKey{},
	},
	{
		ID:          "voice",
		Name:        "Voice",
		Description: "Voice call and audio processing",
		Category:    "tool",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/voice",
		ConfigKeys: []ConfigKey{
			{Key: "elevenlabs_key", Name: "ElevenLabs API Key", Description: "ElevenLabs API Key for TTS", Required: false, Secret: true},
		},
	},
	{
		ID:          "canvas",
		Name:        "Canvas",
		Description: "Visual workspace for real-time collaboration",
		Category:    "tool",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/canvas",
		ConfigKeys:  []ConfigKey{},
	},
	{
		ID:          "calendar",
		Name:        "Calendar",
		Description: "Calendar integration (Google, Apple)",
		Category:    "skill",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/calendar",
		ConfigKeys: []ConfigKey{
			{Key: "provider", Name: "Provider", Description: "Calendar provider (google, apple)", Required: true, Secret: false},
			{Key: "credentials", Name: "Credentials", Description: "OAuth credentials JSON", Required: true, Secret: true},
		},
	},
	{
		ID:          "email",
		Name:        "Email",
		Description: "Email client integration",
		Category:    "skill",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/email",
		ConfigKeys: []ConfigKey{
			{Key: "imap_host", Name: "IMAP Host", Description: "IMAP server hostname", Required: true, Secret: false},
			{Key: "imap_user", Name: "IMAP User", Description: "IMAP username", Required: true, Secret: false},
			{Key: "imap_pass", Name: "IMAP Password", Description: "IMAP password", Required: true, Secret: true},
			{Key: "smtp_host", Name: "SMTP Host", Description: "SMTP server hostname", Required: false, Secret: false},
			{Key: "smtp_user", Name: "SMTP User", Description: "SMTP username", Required: false, Secret: false},
			{Key: "smtp_pass", Name: "SMTP Password", Description: "SMTP password", Required: false, Secret: true},
		},
	},
	{
		ID:          "signal",
		Name:        "Signal",
		Description: "Signal messaging integration",
		Category:    "channel",
		Version:     "1.0.0",
		Author:      "OpenClaw",
		Repository:  "https://github.com/openclaw/plugins/signal",
		ConfigKeys: []ConfigKey{
			{Key: "phone_number", Name: "Phone Number", Description: "Signal phone number", Required: true, Secret: false},
		},
	},
}

func GetBundledPluginDefinition(id string) *BundledPlugin {
	for i := range BundledDefinitions {
		if BundledDefinitions[i].ID == id {
			return &BundledDefinitions[i]
		}
	}
	return nil
}

func GetBundledPluginsByCategory(category string) []BundledPlugin {
	var result []BundledPlugin
	for i := range BundledDefinitions {
		if BundledDefinitions[i].Category == category {
			result = append(result, BundledDefinitions[i])
		}
	}
	return result
}
