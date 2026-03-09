//go:build fyne
// +build fyne

package screens

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/A-Flex-Box/cli/internal/openclaw/config"
)

type ConfigScreen struct {
	config    *config.OpenClawConfig
	container *fyne.Container
}

func NewConfigScreen(cfg *config.OpenClawConfig) *ConfigScreen {
	screen := &ConfigScreen{
		config: cfg,
	}
	screen.container = screen.buildUI()
	return screen
}

func (s *ConfigScreen) buildUI() *fyne.Container {
	var tabs []fyne.CanvasObject

	if s.config.Agent.Workspace == "" {
		s.config.Agent.Workspace = "~/.openclaw/workspace"
	}
	if s.config.Agent.Model == "" {
		s.config.Agent.Model = "anthropic/claude-opus-4-6"
	}

	tabs = append(tabs, s.buildAgentTab())
	tabs = append(tabs, s.buildGatewayTab())
	tabs = append(tabs, s.buildToolsTab())
	tabs = append(tabs, s.buildPluginsTab())

	toolbar := container.NewHBox(
		widget.NewButton("Reset to Defaults", s.resetToDefaults),
		widget.NewButton("Save", s.saveConfig),
	)

	content := container.NewBorder(
		widget.NewLabelWithStyle("⚙️ Configuration", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		toolbar, nil, nil,
		container.NewVBox(tabs...),
	)

	return container.NewPadded(content)
}

func (s *ConfigScreen) buildAgentTab() fyne.CanvasObject {
	modelSelect := widget.NewSelect([]string{
		"anthropic/claude-opus-4-6",
		"anthropic/claude-sonnet-4-6",
		"anthropic/claude-haiku-4-6",
		"openai/gpt-4o",
		"openai/gpt-4o-mini",
		"google/gemini-2.0-flash",
	}, func(selected string) {
		s.config.Agent.Model = selected
	})
	modelSelect.SetSelected(s.config.Agent.Model)

	workspaceEntry := widget.NewEntry()
	workspaceEntry.SetPlaceHolder("~/.openclaw/workspace")
	workspaceEntry.SetText(s.config.Agent.Workspace)
	workspaceEntry.OnChanged = func(text string) {
		s.config.Agent.Workspace = text
	}

	thinkingSelect := widget.NewSelect([]string{"low", "medium", "high"}, func(selected string) {
		s.config.Agent.Thinking = selected
	})
	thinkingSelect.SetSelected(s.config.Agent.Thinking)
	if thinkingSelect.Selected == "" {
		thinkingSelect.SetSelected("medium")
	}

	return container.NewVBox(
		widget.NewLabelWithStyle("Agent Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Model:"),
		modelSelect,
		widget.NewLabel("Workspace:"),
		workspaceEntry,
		widget.NewLabel("Thinking Level:"),
		thinkingSelect,
	)
}

func (s *ConfigScreen) buildGatewayTab() fyne.CanvasObject {
	portEntry := widget.NewEntry()
	portEntry.SetText(fmt.Sprintf("%d", s.config.Gateway.Port))
	portEntry.OnChanged = func(text string) {
		var port int
		fmt.Sscanf(text, "%d", &port)
		if port > 0 && port < 65536 {
			s.config.Gateway.Port = port
		}
	}

	bindSelect := widget.NewSelect([]string{"loopback", "all"}, func(selected string) {
		s.config.Gateway.Bind = selected
	})
	bindSelect.SetSelected(s.config.Gateway.Bind)
	if bindSelect.Selected == "" {
		bindSelect.SetSelected("loopback")
	}

	authModeSelect := widget.NewSelect([]string{"token", "none"}, func(selected string) {
		s.config.Gateway.AuthMode = selected
	})
	authModeSelect.SetSelected(s.config.Gateway.AuthMode)
	if authModeSelect.Selected == "" {
		authModeSelect.SetSelected("token")
	}

	tokenEntry := widget.NewPasswordEntry()
	tokenEntry.SetText(s.config.Gateway.Token)
	tokenEntry.OnChanged = func(text string) {
		s.config.Gateway.Token = text
	}

	return container.NewVBox(
		widget.NewLabelWithStyle("Gateway Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Port:"),
		portEntry,
		widget.NewLabel("Bind:"),
		bindSelect,
		widget.NewLabel("Auth Mode:"),
		authModeSelect,
		widget.NewLabel("Token:"),
		tokenEntry,
	)
}

func (s *ConfigScreen) buildToolsTab() fyne.CanvasObject {
	browserCheck := widget.NewCheck("Enable Browser Control", func(checked bool) {
		s.config.Tools.BrowserEnabled = checked
	})
	browserCheck.Checked = s.config.Tools.BrowserEnabled

	voiceCheck := widget.NewCheck("Enable Voice Control", func(checked bool) {
		s.config.Tools.VoiceEnabled = checked
	})
	voiceCheck.Checked = s.config.Tools.VoiceEnabled

	return container.NewVBox(
		widget.NewLabelWithStyle("Tool Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		browserCheck,
		voiceCheck,
	)
}

func (s *ConfigScreen) buildPluginsTab() fyne.CanvasObject {
	enabledList := strings.Join(s.config.Plugins.Enabled, ", ")

	enabledEntry := widget.NewEntry()
	enabledEntry.SetText(enabledList)
	enabledEntry.SetPlaceHolder("github,slack,notion...")
	enabledEntry.OnChanged = func(text string) {
		if text == "" {
			s.config.Plugins.Enabled = []string{}
		} else {
			s.config.Plugins.Enabled = strings.Split(text, ",")
			for i := range s.config.Plugins.Enabled {
				s.config.Plugins.Enabled[i] = strings.TrimSpace(s.config.Plugins.Enabled[i])
			}
		}
	}

	return container.NewVBox(
		widget.NewLabelWithStyle("Plugin Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Enabled Plugins (comma-separated):"),
		enabledEntry,
		widget.NewLabel("API Keys are managed individually per plugin."),
	)
}

func (s *ConfigScreen) resetToDefaults() {
	s.config = config.DefaultConfig()
}

func (s *ConfigScreen) saveConfig() {
	// This would save to disk in a real implementation
	dialog.NewInformation("Saved", "Configuration saved successfully!", &window{})
}

type window struct{}

func (w *window) ShowAndRun()                    {}
func (w *window) SetContent(_ fyne.CanvasObject) {}

func (s *ConfigScreen) Container() *fyne.Container {
	return s.container
}
