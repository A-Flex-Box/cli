//go:build fyne
// +build fyne

package screens

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/A-Flex-Box/cli/internal/openclaw/service"
)

type StatusScreen struct {
	manager   *service.Manager
	container *fyne.Container
	status    *widget.Label
	uptime    *widget.Label
	port      *widget.Label
	pid       *widget.Label
	memory    *widget.Label
	version   *widget.Label
	stopBtn   *widget.Button
	startBtn  *widget.Button
}

func NewStatusScreen(manager *service.Manager) *StatusScreen {
	screen := &StatusScreen{
		manager: manager,
	}
	screen.container = screen.buildUI()
	screen.refresh()
	return screen
}

func (s *StatusScreen) buildUI() *fyne.Container {
	s.status = widget.NewLabel("● Unknown")
	s.uptime = widget.NewLabel("Uptime: -")
	s.port = widget.NewLabel("Port: 18789")
	s.pid = widget.NewLabel("PID: -")
	s.memory = widget.NewLabel("Memory: -")
	s.version = widget.NewLabel("Version: -")

	s.status.Importance = widget.HighImportance

	statusCard := container.NewVBox(
		widget.NewLabelWithStyle("Gateway Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		s.status,
		s.uptime,
		s.port,
		s.pid,
		s.memory,
		s.version,
	)

	s.stopBtn = widget.NewButton("Stop", func() {
		if err := s.manager.Stop(context.Background()); err != nil {
			s.status.SetText(fmt.Sprintf("Error: %v", err))
		} else {
			s.refresh()
		}
	})

	s.startBtn = widget.NewButton("Start", func() {
		if err := s.manager.Start(context.Background()); err != nil {
			s.status.SetText(fmt.Sprintf("Error: %v", err))
		} else {
			s.refresh()
		}
	})

	restartBtn := widget.NewButton("Restart", func() {
		if err := s.manager.Restart(context.Background()); err != nil {
			s.status.SetText(fmt.Sprintf("Error: %v", err))
		} else {
			s.refresh()
		}
	})

	refreshBtn := widget.NewButton("Refresh", func() {
		s.refresh()
	})

	buttons := container.NewHBox(s.startBtn, s.stopBtn, restartBtn, refreshBtn)

	statsCard := container.NewVBox(
		widget.NewLabelWithStyle("Statistics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Requests Today: -"),
		widget.NewLabel("Total Requests: -"),
		widget.NewLabel("Avg Response: -"),
		widget.NewLabel("Tokens Used: -"),
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle("📊 OpenClaw Status", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		statusCard,
		widget.NewSeparator(),
		buttons,
		widget.NewSeparator(),
		statsCard,
	)

	return container.NewPadded(content)
}

func (s *StatusScreen) refresh() {
	info := s.manager.Status()

	switch info.Status {
	case service.StatusRunning:
		s.status.SetText("● Running")
		s.status.Importance = widget.HighImportance
		s.stopBtn.Enable()
		s.startBtn.Disable()
	case service.StatusStopped:
		s.status.SetText("○ Stopped")
		s.status.Importance = widget.MediumImportance
		s.stopBtn.Disable()
		s.startBtn.Enable()
	default:
		s.status.SetText("○ Unknown")
		s.status.Importance = widget.LowImportance
	}

	s.port.SetText(fmt.Sprintf("Port: %d", info.Port))

	if info.PID > 0 {
		s.pid.SetText(fmt.Sprintf("PID: %d", info.PID))
	}

	if info.Uptime != "" {
		s.uptime.SetText(fmt.Sprintf("Uptime: %s", info.Uptime))
	}

	if info.Memory != "" {
		s.memory.SetText(fmt.Sprintf("Memory: %s", info.Memory))
	}

	if info.Version != "" {
		s.version.SetText(fmt.Sprintf("Version: %s", info.Version))
	}
}

func (s *StatusScreen) Container() *fyne.Container {
	return s.container
}

func (s *StatusScreen) StartAutoRefresh(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.refresh()
		}
	}()
}
