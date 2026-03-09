//go:build fyne
// +build fyne

package widgets

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ProgressWidget struct {
	widget.BaseWidget
	bar     *widget.ProgressBar
	stage   *widget.Label
	message *widget.Label
}

func NewProgressWidget() *ProgressWidget {
	pw := &ProgressWidget{
		bar:     widget.NewProgressBar(),
		stage:   widget.NewLabel("Ready"),
		message: widget.NewLabel(""),
	}
	pw.ExtendBaseWidget(pw)
	return pw
}

func (pw *ProgressWidget) SetProgress(current, total int) {
	if total > 0 {
		pw.bar.SetValue(float64(current) / float64(total))
	} else {
		pw.bar.SetValue(0)
	}
}

func (pw *ProgressWidget) SetStage(stage string) {
	pw.stage.SetText(stage)
}

func (pw *ProgressWidget) SetMessage(message string) {
	pw.message.SetText(message)
}

func (pw *ProgressWidget) Update(stage string, current, total int, message string) {
	pw.SetStage(stage)
	pw.SetProgress(current, total)
	pw.SetMessage(message)
}

func (pw *ProgressWidget) Reset() {
	pw.bar.SetValue(0)
	pw.stage.SetText("Ready")
	pw.message.SetText("")
}

func (pw *ProgressWidget) Complete(message string) {
	pw.bar.SetValue(1)
	pw.stage.SetText("Complete")
	pw.message.SetText(message)
}

func (pw *ProgressWidget) Error(err error) {
	pw.stage.SetText("Error")
	pw.message.SetText(fmt.Sprintf("Error: %v", err))
}

func (pw *ProgressWidget) CreateRenderer() fyne.WidgetRenderer {
	content := container.NewVBox(
		pw.stage,
		pw.bar,
		pw.message,
	)
	return widget.NewSimpleRenderer(content)
}

type StepProgress struct {
	widget.BaseWidget
	steps     []string
	completed int
	container *fyne.Container
}

func NewStepProgress(steps []string) *StepProgress {
	sp := &StepProgress{
		steps:     steps,
		completed: 0,
	}
	sp.ExtendBaseWidget(sp)
	sp.container = sp.buildUI()
	return sp
}

func (sp *StepProgress) buildUI() *fyne.Container {
	var items []fyne.CanvasObject

	for i, step := range sp.steps {
		var icon string
		if i < sp.completed {
			icon = "✓"
		} else if i == sp.completed {
			icon = "●"
		} else {
			icon = "○"
		}

		label := widget.NewLabel(fmt.Sprintf("%s %d. %s", icon, i+1, step))
		items = append(items, label)
	}

	return container.NewVBox(items...)
}

func (sp *StepProgress) SetCompleted(count int) {
	sp.completed = count
	sp.container = sp.buildUI()
	sp.Refresh()
}

func (sp *StepProgress) Next() {
	if sp.completed < len(sp.steps) {
		sp.completed++
		sp.container = sp.buildUI()
		sp.Refresh()
	}
}

func (sp *StepProgress) Reset() {
	sp.completed = 0
	sp.container = sp.buildUI()
	sp.Refresh()
}

func (sp *StepProgress) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(sp.container)
}
