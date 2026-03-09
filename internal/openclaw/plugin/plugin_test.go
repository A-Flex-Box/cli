package plugin

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()

	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	plugins := m.List()
	if len(plugins) == 0 {
		t.Error("expected non-empty plugin list")
	}
}

func TestGetByID(t *testing.T) {
	m := NewManager()

	p := m.GetByID("github")
	if p == nil {
		t.Error("expected to find github plugin")
	}
	if p.Name != "GitHub" {
		t.Errorf("expected name 'GitHub', got %s", p.Name)
	}

	p = m.GetByID("nonexistent")
	if p != nil {
		t.Error("expected nil for nonexistent plugin")
	}
}

func TestInstallUninstall(t *testing.T) {
	m := NewManager()

	if err := m.Install("github"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	p := m.GetByID("github")
	if !p.Installed {
		t.Error("plugin should be installed")
	}
	if !p.Enabled {
		t.Error("plugin should be enabled after install")
	}

	if err := m.Uninstall("github"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	p = m.GetByID("github")
	if p.Installed {
		t.Error("plugin should not be installed")
	}
}

func TestEnableDisable(t *testing.T) {
	m := NewManager()

	if err := m.Enable("slack"); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	p := m.GetByID("slack")
	if !p.Enabled {
		t.Error("plugin should be enabled")
	}

	if err := m.Disable("slack"); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	p = m.GetByID("slack")
	if p.Enabled {
		t.Error("plugin should be disabled")
	}
}

func TestListByCategory(t *testing.T) {
	m := NewManager()

	channels := m.ListByCategory("channel")
	if len(channels) == 0 {
		t.Error("expected non-empty channel list")
	}

	for _, p := range channels {
		if p.Category != "channel" {
			t.Errorf("expected channel category, got %s", p.Category)
		}
	}
}

func TestGetBundledPluginDefinition(t *testing.T) {
	def := GetBundledPluginDefinition("github")
	if def == nil {
		t.Fatal("expected github plugin definition")
	}

	if def.Name != "GitHub" {
		t.Errorf("expected name 'GitHub', got %s", def.Name)
	}

	if len(def.ConfigKeys) == 0 {
		t.Error("expected github to have config keys")
	}

	foundToken := false
	for _, key := range def.ConfigKeys {
		if key.Key == "token" {
			foundToken = true
			if !key.Secret {
				t.Error("token should be marked as secret")
			}
		}
	}
	if !foundToken {
		t.Error("expected github to have token config key")
	}
}

func TestGetBundledPluginsByCategory(t *testing.T) {
	skills := GetBundledPluginsByCategory("skill")
	if len(skills) == 0 {
		t.Error("expected non-empty skills list")
	}

	for _, s := range skills {
		if s.Category != "skill" {
			t.Errorf("expected skill category, got %s", s.Category)
		}
	}
}
