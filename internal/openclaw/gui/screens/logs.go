//go:build fyne
// +build fyne

package screens

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/A-Flex-Box/cli/internal/openclaw/service"
)

type LogScreen struct {
	manager   *service.Manager
	container *fyne.Container
	logEntry  *widget.Entry
	logs      []string
	lineCount int
}

func NewLogScreen(manager *service.Manager) *LogScreen {
	screen := &LogScreen{
		manager:   manager,
		lineCount: 100,
	}
	screen.container = screen.buildUI()
	return screen
}

func (s *LogScreen) buildUI() *fyne.Container {
	s.logEntry = widget.NewMultiLineEntry()
	s.logEntry.SetPlaceHolder("Logs will appear here...")
	s.logEntry.Disable()

	linesSelect := widget.NewSelect([]string{"50", "100", "200", "500"}, func(selected string) {
		switch selected {
		case "50":
			s.lineCount = 50
		case "100":
			s.lineCount = 100
		case "200":
			s.lineCount = 200
		case "500":
			s.lineCount = 500
		}
		s.refresh()
	})
	linesSelect.SetSelected("100")

	refreshBtn := widget.NewButton("Refresh", func() {
		s.refresh()
	})

	tailBtn := widget.NewButton("Tail (Follow)", func() {
		s.showTailModeInfo()
	})

	clearBtn := widget.NewButton("Clear", func() {
		s.logs = nil
		s.logEntry.SetText("")
	})

	exportBtn := widget.NewButton("Export...", func() {
		// TODO: Implement file export
	})

	toolbar := container.NewHBox(
		widget.NewLabel("Lines:"),
		linesSelect,
		refreshBtn,
		tailBtn,
		clearBtn,
		exportBtn,
	)

	levelFilter := widget.NewSelect([]string{"All", "INFO", "WARN", "ERROR", "DEBUG"}, func(selected string) {
		s.filterByLevel(selected)
	})
	levelFilter.SetSelected("All")

	filterBar := container.NewHBox(
		widget.NewLabel("Level:"),
		levelFilter,
	)

	scroll := container.NewScroll(s.logEntry)
	scroll.SetMinSize(fyne.NewSize(700, 400))

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("📋 Gateway Logs", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			toolbar,
			filterBar,
		),
		nil, nil, nil,
		scroll,
	)

	return content
}

func (s *LogScreen) refresh() {
	logs, err := s.manager.GetLogs(s.lineCount)
	if err != nil {
		s.logEntry.SetText(fmt.Sprintf("Error reading logs: %v", err))
		return
	}
	s.logs = logs
	s.logEntry.SetText(strings.Join(logs, "\n"))
}

func (s *LogScreen) filterByLevel(level string) {
	if level == "All" {
		s.logEntry.SetText(strings.Join(s.logs, "\n"))
		return
	}

	var filtered []string
	for _, log := range s.logs {
		if strings.Contains(strings.ToUpper(log), level) {
			filtered = append(filtered, log)
		}
	}
	s.logEntry.SetText(strings.Join(filtered, "\n"))
}

func (s *LogScreen) showTailModeInfo() {
	dialog := widget.NewPopUp(
		container.NewVBox(
			widget.NewLabel("Tail Mode"),
			widget.NewLabel("Follow mode tails the log file in real-time."),
			widget.NewLabel("(Feature coming soon)"),
		),
		&fyne.Container{},
	)
	dialog.Show()
}

func (s *LogScreen) Container() *fyne.Container {
	return s.container
}
