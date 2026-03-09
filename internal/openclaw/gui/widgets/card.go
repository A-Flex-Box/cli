//go:build fyne
// +build fyne

package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Card struct {
	widget.BaseWidget
	title   string
	content fyne.CanvasObject
}

func NewCard(title string, content fyne.CanvasObject) *Card {
	c := &Card{
		title:   title,
		content: content,
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *Card) CreateRenderer() fyne.WidgetRenderer {
	titleLabel := widget.NewLabelWithStyle(c.title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	box := container.NewVBox(titleLabel, widget.NewSeparator(), c.content)
	return widget.NewSimpleRenderer(box)
}
