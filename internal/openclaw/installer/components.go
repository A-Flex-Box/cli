package installer

import (
	"fmt"
	"sort"
)

type ComponentCategory string

const (
	CategoryCore    ComponentCategory = "core"
	CategoryChannel ComponentCategory = "channel"
	CategoryTool    ComponentCategory = "tool"
	CategorySkill   ComponentCategory = "skill"
)

type Component struct {
	ID           string
	Name         string
	Description  string
	Category     ComponentCategory
	Dependencies []string
	Conflicts    []string
	Required     []string
	Optional     bool
	InstallSize  int64
	Platform     []string
}

type ComponentList struct {
	Items    []Component
	Selected map[string]bool
}

var DefaultComponents = []Component{
	{
		ID:          "core",
		Name:        "OpenClaw Core",
		Description: "Core runtime and agent framework",
		Category:    CategoryCore,
		Optional:    false,
		InstallSize: 45 * 1024 * 1024,
	},
	{
		ID:           "gateway",
		Name:         "Gateway Service",
		Description:  "HTTP API, WebSocket server and control UI",
		Category:     CategoryCore,
		Dependencies: []string{"core"},
		Optional:     false,
		InstallSize:  15 * 1024 * 1024,
	},

	{
		ID:           "whatsapp",
		Name:         "WhatsApp",
		Description:  "WhatsApp messaging integration",
		Category:     CategoryChannel,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  5 * 1024 * 1024,
	},
	{
		ID:           "telegram",
		Name:         "Telegram",
		Description:  "Telegram bot integration",
		Category:     CategoryChannel,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  3 * 1024 * 1024,
	},
	{
		ID:           "slack",
		Name:         "Slack",
		Description:  "Slack workspace integration",
		Category:     CategoryChannel,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  4 * 1024 * 1024,
	},
	{
		ID:           "discord",
		Name:         "Discord",
		Description:  "Discord bot integration",
		Category:     CategoryChannel,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  4 * 1024 * 1024,
	},
	{
		ID:           "imessage",
		Name:         "iMessage",
		Description:  "iMessage integration (macOS only)",
		Category:     CategoryChannel,
		Dependencies: []string{"core"},
		Required:     []string{"darwin"},
		Optional:     true,
		Platform:     []string{"darwin"},
		InstallSize:  8 * 1024 * 1024,
	},
	{
		ID:           "signal",
		Name:         "Signal",
		Description:  "Signal messaging integration",
		Category:     CategoryChannel,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  4 * 1024 * 1024,
	},

	{
		ID:           "browser",
		Name:         "Browser Control",
		Description:  "Web browser automation and control",
		Category:     CategoryTool,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  10 * 1024 * 1024,
	},
	{
		ID:           "voice",
		Name:         "Voice Call",
		Description:  "Voice call and audio processing",
		Category:     CategoryTool,
		Dependencies: []string{"core", "browser"},
		Optional:     true,
		InstallSize:  12 * 1024 * 1024,
	},
	{
		ID:           "canvas",
		Name:         "Canvas",
		Description:  "Visual workspace for real-time collaboration",
		Category:     CategoryTool,
		Dependencies: []string{"core", "browser"},
		Optional:     true,
		InstallSize:  8 * 1024 * 1024,
	},

	{
		ID:           "github",
		Name:         "GitHub",
		Description:  "GitHub repository and issue management",
		Category:     CategorySkill,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  2 * 1024 * 1024,
	},
	{
		ID:           "notion",
		Name:         "Notion",
		Description:  "Notion workspace integration",
		Category:     CategorySkill,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  2 * 1024 * 1024,
	},
	{
		ID:           "obsidian",
		Name:         "Obsidian",
		Description:  "Obsidian notes integration",
		Category:     CategorySkill,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  2 * 1024 * 1024,
	},
	{
		ID:           "spotify",
		Name:         "Spotify Player",
		Description:  "Spotify music control",
		Category:     CategorySkill,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  3 * 1024 * 1024,
	},
	{
		ID:           "weather",
		Name:         "Weather",
		Description:  "Weather information lookup",
		Category:     CategorySkill,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  1 * 1024 * 1024,
	},
	{
		ID:           "calendar",
		Name:         "Calendar",
		Description:  "Calendar integration (Google, Apple)",
		Category:     CategorySkill,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  2 * 1024 * 1024,
	},
	{
		ID:           "email",
		Name:         "Email",
		Description:  "Email client integration",
		Category:     CategorySkill,
		Dependencies: []string{"core"},
		Optional:     true,
		InstallSize:  3 * 1024 * 1024,
	},
}

func NewComponentList() *ComponentList {
	return &ComponentList{
		Items:    DefaultComponents,
		Selected: make(map[string]bool),
	}
}

func (cl *ComponentList) GetByID(id string) *Component {
	for i := range cl.Items {
		if cl.Items[i].ID == id {
			return &cl.Items[i]
		}
	}
	return nil
}

func (cl *ComponentList) Select(id string) {
	comp := cl.GetByID(id)
	if comp != nil {
		cl.Selected[id] = true
		for _, dep := range comp.Dependencies {
			cl.Select(dep)
		}
	}
}

func (cl *ComponentList) Deselect(id string) {
	comp := cl.GetByID(id)
	if comp == nil {
		return
	}
	for cid := range cl.Selected {
		other := cl.GetByID(cid)
		if other != nil {
			for _, dep := range other.Dependencies {
				if dep == id {
					cl.Deselect(cid)
				}
			}
		}
	}
	delete(cl.Selected, id)
}

func (cl *ComponentList) IsSelected(id string) bool {
	return cl.Selected[id]
}

func (cl *ComponentList) GetSelected() []Component {
	var selected []Component
	for id := range cl.Selected {
		if comp := cl.GetByID(id); comp != nil {
			selected = append(selected, *comp)
		}
	}
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].ID < selected[j].ID
	})
	return selected
}

func (cl *ComponentList) GetByCategory(cat ComponentCategory) []Component {
	var result []Component
	for _, comp := range cl.Items {
		if comp.Category == cat {
			result = append(result, comp)
		}
	}
	return result
}

func (cl *ComponentList) GetCategories() []ComponentCategory {
	return []ComponentCategory{CategoryCore, CategoryChannel, CategoryTool, CategorySkill}
}

func (cl *ComponentList) Validate(currentOS string) []error {
	var errors []error
	for id := range cl.Selected {
		comp := cl.GetByID(id)
		if comp == nil {
			errors = append(errors, fmt.Errorf("unknown component: %s", id))
			continue
		}
		if len(comp.Platform) > 0 {
			platformOK := false
			for _, p := range comp.Platform {
				if p == currentOS {
					platformOK = true
					break
				}
			}
			if !platformOK {
				errors = append(errors, fmt.Errorf("component %s is not available on %s", comp.Name, currentOS))
			}
		}
		for _, conflict := range comp.Conflicts {
			if cl.Selected[conflict] {
				errors = append(errors, fmt.Errorf("component %s conflicts with %s", comp.Name, conflict))
			}
		}
	}
	return errors
}

func (cl *ComponentList) TotalSize() int64 {
	var total int64
	for id := range cl.Selected {
		if comp := cl.GetByID(id); comp != nil {
			total += comp.InstallSize
		}
	}
	return total
}

func (cl *ComponentList) SelectDefaults(currentOS string) {
	for _, comp := range cl.Items {
		if !comp.Optional {
			if len(comp.Platform) == 0 {
				cl.Select(comp.ID)
			} else {
				for _, p := range comp.Platform {
					if p == currentOS {
						cl.Select(comp.ID)
						break
					}
				}
			}
		}
	}
}

func ResolveDependencies(componentIDs []string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		for _, comp := range DefaultComponents {
			if comp.ID == id {
				for _, dep := range comp.Dependencies {
					visit(dep)
				}
				break
			}
		}
		result = append(result, id)
	}

	for _, id := range componentIDs {
		visit(id)
	}

	return result
}
