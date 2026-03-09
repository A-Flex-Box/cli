package installer

import (
	"testing"
)

func TestNewComponentList(t *testing.T) {
	cl := NewComponentList()

	if len(cl.Items) == 0 {
		t.Error("expected non-empty component list")
	}

	for _, comp := range cl.Items {
		if comp.ID == "" {
			t.Error("component has empty ID")
		}
		if comp.Name == "" {
			t.Error("component has empty Name")
		}
	}
}

func TestComponentSelect(t *testing.T) {
	cl := NewComponentList()

	cl.Select("github")

	if !cl.IsSelected("github") {
		t.Error("github should be selected")
	}
	if !cl.IsSelected("core") {
		t.Error("core should be auto-selected as dependency")
	}
}

func TestComponentDeselect(t *testing.T) {
	cl := NewComponentList()

	cl.Select("core")
	cl.Select("github")

	cl.Deselect("core")

	if cl.IsSelected("core") {
		t.Error("core should be deselected")
	}
	if cl.IsSelected("github") {
		t.Error("github should be auto-deselected when dependency removed")
	}
}

func TestComponentGetSelected(t *testing.T) {
	cl := NewComponentList()

	cl.Select("github")
	cl.Select("slack")

	selected := cl.GetSelected()

	if len(selected) < 3 {
		t.Errorf("expected at least 3 selected (github, slack, core), got %d", len(selected))
	}
}

func TestComponentGetByCategory(t *testing.T) {
	cl := NewComponentList()

	skills := cl.GetByCategory(CategorySkill)
	if len(skills) == 0 {
		t.Error("expected non-empty skill list")
	}

	for _, s := range skills {
		if s.Category != CategorySkill {
			t.Errorf("expected skill category, got %s", s.Category)
		}
	}
}

func TestComponentTotalSize(t *testing.T) {
	cl := NewComponentList()
	cl.SelectDefaults("linux")

	size := cl.TotalSize()
	if size == 0 {
		t.Error("expected non-zero total size for default selection")
	}
}

func TestResolveDependencies(t *testing.T) {
	deps := ResolveDependencies([]string{"voice"})

	hasCore := false
	hasBrowser := false
	hasVoice := false

	for _, dep := range deps {
		switch dep {
		case "core":
			hasCore = true
		case "browser":
			hasBrowser = true
		case "voice":
			hasVoice = true
		}
	}

	if !hasCore {
		t.Error("core dependency not resolved")
	}
	if !hasBrowser {
		t.Error("browser dependency not resolved")
	}
	if !hasVoice {
		t.Error("voice not in resolved list")
	}
}

func TestValidatePlatform(t *testing.T) {
	cl := NewComponentList()

	cl.Select("imessage")

	errs := cl.Validate("linux")
	if len(errs) == 0 {
		t.Error("expected validation error for imessage on linux")
	}

	errs = cl.Validate("darwin")
	if len(errs) != 0 {
		t.Errorf("expected no validation errors for imessage on darwin, got %d", len(errs))
	}
}

func TestSelectDefaults(t *testing.T) {
	cl := NewComponentList()
	cl.SelectDefaults("linux")

	if !cl.IsSelected("core") {
		t.Error("core should be selected by default")
	}
	if !cl.IsSelected("gateway") {
		t.Error("gateway should be selected by default")
	}
}
