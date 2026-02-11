package doctor

import (
	"fmt"
	"os"
	"time"

	"github.com/A-Flex-Box/cli/internal/doctor"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// sparklineChars for ASCII sparkline (▁▂▃▄▅▆▇█)
var sparklineChars = []rune("▁▂▃▄▅▆▇█")

const watchInterval = time.Second

type watchModel struct {
	viewport     viewport.Model
	interfaces   []doctor.InterfaceStat
	prevStats    map[string]doctor.InterfaceStat
	connections  []doctor.ConnectionInfo
	sparkRecv    []float64
	sparkSent    []float64
	maxSparkLen  int
	err          error
}

type tickMsg struct{}

func newWatchModel() watchModel {
	vp := viewport.New(80, 24)
	vp.Style = lipgloss.NewStyle()
	return watchModel{
		viewport:    vp,
		prevStats:   make(map[string]doctor.InterfaceStat),
		maxSparkLen: 30,
	}
}

func (m watchModel) Init() tea.Cmd {
	logger.Info("doctor watch Init",
		zap.String("component", "cmd.doctor.watch"))
	return tea.Batch(m.tick(), viewport.Sync(m.viewport))
}

func (m watchModel) tick() tea.Cmd {
	return tea.Tick(watchInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m watchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			logger.Info("doctor watch quit",
				zap.String("component", "cmd.doctor.watch"))
			return m, tea.Quit
		}
	case tickMsg:
		var cmd tea.Cmd
		m, cmd = m.refresh()
		return m, tea.Batch(cmd, m.tick())
	}
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, vpCmd
}

func (m *watchModel) refresh() (watchModel, tea.Cmd) {
	ifs, err := doctor.GetInterfaceStats()
	if err != nil {
		logger.Warn("doctor watch GetInterfaceStats failed",
			zap.String("component", "cmd.doctor.watch"),
			zap.Error(err))
		m.err = err
		return *m, nil
	}
	m.interfaces = ifs

	// Compute deltas for sparkline
	nowRecv, nowSent := uint64(0), uint64(0)
	for _, s := range ifs {
		nowRecv += s.BytesRecv
		nowSent += s.BytesSent
	}
	prevRecv, prevSent := uint64(0), uint64(0)
	for _, s := range m.prevStats {
		prevRecv += s.BytesRecv
		prevSent += s.BytesSent
	}
	deltaRecv := float64(nowRecv - prevRecv)
	deltaSent := float64(nowSent - prevSent)
	// Normalize to 0-7 for sparkline char index
	m.sparkRecv = append(m.sparkRecv, deltaRecv)
	m.sparkSent = append(m.sparkSent, deltaSent)
	if len(m.sparkRecv) > m.maxSparkLen {
		m.sparkRecv = m.sparkRecv[1:]
		m.sparkSent = m.sparkSent[1:]
	}
	for i, s := range ifs {
		m.prevStats[s.Name] = s
		_ = i
	}

	conns, err := doctor.GetActiveConnections(15)
	if err != nil {
		logger.Debug("doctor watch GetActiveConnections failed",
			zap.String("component", "cmd.doctor.watch"),
			zap.Error(err))
	} else {
		m.connections = conns
	}

	m.viewport.SetContent(m.renderContent())
	return *m, nil
}

func (m watchModel) renderContent() string {
	var b string
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render("Network Watch Dashboard")
	b += title + "\n\n"

	// Sparkline (Recv/Sent)
	b += lipgloss.NewStyle().Foreground(lipgloss.Color("#43BF6D")).Render("I/O Sparkline (Recv) ") + m.renderSparkline(m.sparkRecv) + "\n"
	b += lipgloss.NewStyle().Foreground(lipgloss.Color("#F25D94")).Render("I/O Sparkline (Sent) ") + m.renderSparkline(m.sparkSent) + "\n\n"

	// Interfaces table
	b += lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B")).Render("Interfaces") + "\n"
	rows := make([][]string, 0, len(m.interfaces))
	for _, s := range m.interfaces {
		rows = append(rows, []string{
			s.Name,
			fmt.Sprintf("%d", s.BytesRecv),
			fmt.Sprintf("%d", s.BytesSent),
		})
	}
	if len(rows) > 0 {
		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))).
			Headers("Name", "Bytes Recv", "Bytes Sent").
			Rows(rows...).
			Width(60)
		b += t.String() + "\n\n"
	}

	// Connections table
	b += lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B")).Render("Top Connections") + "\n"
	connRows := make([][]string, 0, len(m.connections))
	for _, c := range m.connections {
		connRows = append(connRows, []string{
			c.LocalAddr,
			c.RemoteAddr,
			c.Status,
			fmt.Sprintf("%d", c.PID),
		})
	}
	if len(connRows) > 0 {
		t2 := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))).
			Headers("Local", "Remote", "Status", "PID").
			Rows(connRows...).
			Width(70)
		b += t2.String()
	} else {
		b += "  (no active connections)"
	}

	if m.err != nil {
		b += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#F25D94")).Render("Error: "+m.err.Error())
	}
	return b
}

func (m watchModel) renderSparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	max := 0.0
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	out := make([]rune, len(values))
	for i, v := range values {
		idx := 0
		if max > 0 {
			idx = int((v / max) * 7)
			if idx > 7 {
				idx = 7
			}
		}
		out[i] = sparklineChars[idx]
	}
	return string(out)
}

func (m watchModel) View() string {
	return m.viewport.View()
}

func newWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "watch",
		Short:   "Real-time network dashboard (TUI)",
		Long:    `Full-screen Bubble Tea TUI: interface stats, sparklines, active connections. Refresh every 1s.`,
		Example: "cli doctor watch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch()
		},
	}
}

func runWatch() error {
	logger.Info("doctor watch started",
		zap.String("component", "cmd.doctor.watch"))

	m := newWatchModel()
	// Initial refresh
	_, _ = m.refresh()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		logger.Error("doctor watch tea program error",
			zap.String("component", "cmd.doctor.watch"),
			zap.Error(err))
		os.Exit(1)
	}
	return nil
}
