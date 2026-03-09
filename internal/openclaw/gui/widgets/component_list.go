//go:build fyne
// +build fyne

package widgets

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
)

type ComponentListWidget struct {
	widget.BaseWidget
	components  []installer.Component
	selected    map[string]bool
	onToggle    func(id string, selected bool)
	container   *fyne.Container
	checkStates map[string]*widget.Check
}

func NewComponentListWidget(components []installer.Component) *ComponentListWidget {
	clw := &ComponentListWidget{
		components:  components,
		selected:    make(map[string]bool),
		checkStates: make(map[string]*widget.Check),
	}
	clw.ExtendBaseWidget(clw)
	clw.container = clw.buildUI()
	return clw
}

func (clw *ComponentListWidget) buildUI() *fyne.Container {
	var items []fyne.CanvasObject

	categories := []installer.ComponentCategory{
		installer.CategoryCore,
		installer.CategoryChannel,
		installer.CategoryTool,
		installer.CategorySkill,
	}

	categoryNames := map[installer.ComponentCategory]string{
		installer.CategoryCore:    "Core",
		installer.CategoryChannel: "Channels",
		installer.CategoryTool:    "Tools",
		installer.CategorySkill:   "Skills",
	}

	for _, cat := range categories {
		var categoryItems []fyne.CanvasObject
		categoryItems = append(categoryItems, widget.NewLabelWithStyle(
			categoryNames[cat],
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		))

		for i := range clw.components {
			comp := &clw.components[i]
			if comp.Category != cat {
				continue
			}

			item := clw.createComponentItem(comp)
			categoryItems = append(categoryItems, item)
		}

		if len(categoryItems) > 1 {
			items = append(items, container.NewVBox(categoryItems...))
			items = append(items, widget.NewSeparator())
		}
	}

	return container.NewVBox(items...)
}

func (clw *ComponentListWidget) createComponentItem(comp *installer.Component) fyne.CanvasObject {
	depCount := len(comp.Dependencies)
	depText := ""
	if depCount > 0 {
		depText = fmt.Sprintf(" [%d deps]", depCount)
	}

	sizeText := formatSize(comp.InstallSize)

	label := widget.NewLabel(fmt.Sprintf("%s - %s%s (%s)",
		comp.Name, comp.Description, depText, sizeText))

	check := widget.NewCheck("", func(checked bool) {
		clw.selected[comp.ID] = checked
		if clw.onToggle != nil {
			clw.onToggle(comp.ID, checked)
		}
	})
	check.Checked = !comp.Optional

	clw.checkStates[comp.ID] = check

	return container.NewBorder(nil, nil, check, nil, label)
}

func (clw *ComponentListWidget) SetSelected(id string, selected bool) {
	clw.selected[id] = selected
	if check, ok := clw.checkStates[id]; ok {
		check.Checked = selected
		check.Refresh()
	}
}

func (clw *ComponentListWidget) IsSelected(id string) bool {
	return clw.selected[id]
}

func (clw *ComponentListWidget) GetSelected() []string {
	var result []string
	for id, sel := range clw.selected {
		if sel {
			result = append(result, id)
		}
	}
	return result
}

func (clw *ComponentListWidget) OnToggle(callback func(id string, selected bool)) {
	clw.onToggle = callback
}

func (clw *ComponentListWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(clw.container)
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

type StatusBadge struct {
	widget.BaseWidget
	status string
	color  fyne.CanvasObject
	label  *widget.Label
}

func NewStatusBadge(status string) *StatusBadge {
	sb := &StatusBadge{
		status: status,
		label:  widget.NewLabel(status),
	}
	sb.ExtendBaseWidget(sb)
	sb.updateColor()
	return sb
}

func (sb *StatusBadge) updateColor() {
	var color fyne.CanvasObject
	switch sb.status {
	case "running", "active", "online":
		color = canvas.NewRectangle(colorGreen)
	case "stopped", "inactive", "offline":
		color = canvas.NewRectangle(colorRed)
	default:
		color = canvas.NewRectangle(colorGray)
	}
	sb.color = color
}

func (sb *StatusBadge) SetStatus(status string) {
	sb.status = status
	sb.label.SetText(status)
	sb.updateColor()
	sb.Refresh()
}

func (sb *StatusBadge) CreateRenderer() fyne.WidgetRenderer {
	sb.color.Resize(fyne.NewSize(10, 10))
	return widget.NewSimpleRenderer(
		container.NewHBox(sb.color, sb.label),
	)
}

var (
	colorGreen = fyne.NewColor(0, 128, 0, 255)
	colorRed   = fyne.NewColor(255, 0, 0, 255)
	colorGray  = fyne.NewColor(128, 128, 128, 255)
)
