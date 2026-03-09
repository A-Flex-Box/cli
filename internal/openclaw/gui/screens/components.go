//go:build fyne
// +build fyne

package screens

import (
	"fmt"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
)

type ComponentSelectScreen struct {
	componentList *installer.ComponentList
	currentOS     string
	container     *fyne.Container
	checkBoxes    map[string]*widget.Check
	onChange      func()
}

func NewComponentSelectScreen(currentOS string) *ComponentSelectScreen {
	screen := &ComponentSelectScreen{
		componentList: installer.NewComponentList(),
		currentOS:     currentOS,
		checkBoxes:    make(map[string]*widget.Check),
	}
	screen.componentList.SelectDefaults(currentOS)
	screen.container = screen.buildUI()
	return screen
}

func (s *ComponentSelectScreen) buildUI() *fyne.Container {
	var categoryContainers []fyne.CanvasObject

	categoryNames := map[installer.ComponentCategory]string{
		installer.CategoryCore:    "Core (Required)",
		installer.CategoryChannel: "Channels",
		installer.CategoryTool:    "Tools",
		installer.CategorySkill:   "Skills",
	}

	categoryOrder := []installer.ComponentCategory{
		installer.CategoryCore,
		installer.CategoryChannel,
		installer.CategoryTool,
		installer.CategorySkill,
	}

	for _, cat := range categoryOrder {
		components := s.componentList.GetByCategory(cat)
		if len(components) == 0 {
			continue
		}

		var items []fyne.CanvasObject
		items = append(items, widget.NewLabelWithStyle(
			categoryNames[cat],
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		))

		sort.Slice(components, func(i, j int) bool {
			return components[i].Name < components[j].Name
		})

		for i := range components {
			comp := &components[i]
			box := s.createComponentCheck(comp)
			items = append(items, box)
		}

		categoryContainers = append(categoryContainers, container.NewVBox(items...))
	}

	s.updateStatus()

	selectedLabel := widget.NewLabel("Selected: 0 components")
	sizeLabel := widget.NewLabel("Estimated: 0 MB")
	depsLabel := widget.NewLabel("Dependencies: OK")

	s.onChange = func() {
		selected := s.componentList.GetSelected()
		totalSize := s.componentList.TotalSize()
		errors := s.componentList.Validate(s.currentOS)

		selectedLabel.SetText(fmt.Sprintf("Selected: %d components", len(selected)))
		sizeLabel.SetText(fmt.Sprintf("Estimated: %d MB", totalSize/(1024*1024)))

		if len(errors) > 0 {
			depsLabel.SetText(fmt.Sprintf("Issues: %d", len(errors)))
		} else {
			depsLabel.SetText("Dependencies: OK ✓")
		}
	}

	selectAllBtn := widget.NewButton("Select All Optional", func() {
		for _, comp := range s.componentList.Items {
			if comp.Optional {
				s.componentList.Select(comp.ID)
			}
		}
		s.refreshCheckBoxes()
	})

	clearBtn := widget.NewButton("Clear Optional", func() {
		for id := range s.componentList.Selected {
			comp := s.componentList.GetByID(id)
			if comp != nil && comp.Optional {
				delete(s.componentList.Selected, id)
			}
		}
		s.componentList.SelectDefaults(s.currentOS)
		s.refreshCheckBoxes()
	})

	statusContainer := container.NewHBox(selectedLabel, sizeLabel, depsLabel)

	content := container.NewVBox(
		widget.NewLabelWithStyle("📦 Component Selection", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
	)

	for _, cc := range categoryContainers {
		content.Add(cc)
		content.Add(widget.NewSeparator())
	}

	content.Add(container.NewHBox(selectAllBtn, clearBtn))
	content.Add(widget.NewSeparator())
	content.Add(statusContainer)

	scroll := container.NewScroll(content)
	return container.NewBorder(nil, statusContainer, nil, nil, scroll)
}

func (s *ComponentSelectScreen) createComponentCheck(comp *installer.Component) *widget.Check {
	checked := s.componentList.IsSelected(comp.ID)
	enabled := !comp.Optional

	disabledText := ""
	if len(comp.Platform) > 0 {
		platformOK := false
		for _, p := range comp.Platform {
			if p == s.currentOS {
				platformOK = true
				break
			}
		}
		if !platformOK {
			enabled = false
			disabledText = fmt.Sprintf(" (%s only)", comp.Platform[0])
		}
	}

	depsText := ""
	if len(comp.Dependencies) > 0 {
		depsText = fmt.Sprintf(" [%d deps]", len(comp.Dependencies))
	}

	label := fmt.Sprintf("%s - %s%s%s", comp.Name, comp.Description, depsText, disabledText)

	check := widget.NewCheck(label, func(checked bool) {
		if checked {
			s.componentList.Select(comp.ID)
		} else {
			s.componentList.Deselect(comp.ID)
		}
		s.refreshCheckBoxes()
		if s.onChange != nil {
			s.onChange()
		}
	})
	check.Checked = checked
	check.Disable()

	if comp.Optional && disabledText == "" {
		check.Enable()
	}

	s.checkBoxes[comp.ID] = check
	return check
}

func (s *ComponentSelectScreen) refreshCheckBoxes() {
	for id, check := range s.checkBoxes {
		check.Checked = s.componentList.IsSelected(id)
		check.Refresh()
	}
}

func (s *ComponentSelectScreen) Container() *fyne.Container {
	return s.container
}

func (s *ComponentSelectScreen) GetSelected() []installer.Component {
	return s.componentList.GetSelected()
}

func (s *ComponentSelectScreen) updateStatus() {
}
