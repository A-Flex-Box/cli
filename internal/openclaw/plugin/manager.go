package plugin

type Plugin struct {
	ID          string
	Name        string
	Description string
	Category    string
	Installed   bool
	Enabled     bool
	Config      map[string]interface{}
}

type Manager struct {
	plugins []Plugin
}

func NewManager() *Manager {
	return &Manager{
		plugins: GetBundledPlugins(),
	}
}

func GetBundledPlugins() []Plugin {
	return []Plugin{
		{ID: "github", Name: "GitHub", Description: "GitHub repository integration", Category: "skill"},
		{ID: "slack", Name: "Slack", Description: "Slack workspace integration", Category: "channel"},
		{ID: "discord", Name: "Discord", Description: "Discord bot integration", Category: "channel"},
		{ID: "telegram", Name: "Telegram", Description: "Telegram bot integration", Category: "channel"},
		{ID: "whatsapp", Name: "WhatsApp", Description: "WhatsApp messaging", Category: "channel"},
		{ID: "notion", Name: "Notion", Description: "Notion workspace", Category: "skill"},
		{ID: "obsidian", Name: "Obsidian", Description: "Obsidian notes", Category: "skill"},
		{ID: "spotify", Name: "Spotify", Description: "Spotify music control", Category: "skill"},
		{ID: "weather", Name: "Weather", Description: "Weather information", Category: "skill"},
		{ID: "browser", Name: "Browser", Description: "Browser control", Category: "tool"},
		{ID: "voice", Name: "Voice", Description: "Voice call", Category: "tool"},
		{ID: "canvas", Name: "Canvas", Description: "Visual workspace", Category: "tool"},
	}
}

func (m *Manager) List() []Plugin {
	return m.plugins
}

func (m *Manager) GetByID(id string) *Plugin {
	for i := range m.plugins {
		if m.plugins[i].ID == id {
			return &m.plugins[i]
		}
	}
	return nil
}

func (m *Manager) Install(id string) error {
	for i := range m.plugins {
		if m.plugins[i].ID == id {
			m.plugins[i].Installed = true
			m.plugins[i].Enabled = true
			return nil
		}
	}
	return nil
}

func (m *Manager) Uninstall(id string) error {
	for i := range m.plugins {
		if m.plugins[i].ID == id {
			m.plugins[i].Installed = false
			m.plugins[i].Enabled = false
			return nil
		}
	}
	return nil
}

func (m *Manager) Enable(id string) error {
	for i := range m.plugins {
		if m.plugins[i].ID == id {
			m.plugins[i].Enabled = true
			return nil
		}
	}
	return nil
}

func (m *Manager) Disable(id string) error {
	for i := range m.plugins {
		if m.plugins[i].ID == id {
			m.plugins[i].Enabled = false
			return nil
		}
	}
	return nil
}

func (m *Manager) ListByCategory(category string) []Plugin {
	var result []Plugin
	for _, p := range m.plugins {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}
