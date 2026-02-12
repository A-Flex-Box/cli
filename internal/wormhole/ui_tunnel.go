package wormhole

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxTrafficLogs = 10

// TunnelEventMsg is sent when a UIEvent arrives from the tunnel.
type TunnelEventMsg UIEvent

// tunnelModel holds the Bubble Tea model for the tunnel TUI.
type tunnelModel struct {
	role       string   // "expose" or "connect"
	code       string
	addr       string   // port or bindAddr
	trafficLog []string // last N traffic events, newest last
	width      int
	height     int
	eventsCh   <-chan UIEvent
}

func (m tunnelModel) Init() tea.Cmd {
	return waitForTunnelEvent(m.eventsCh)
}

func (m tunnelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, waitForTunnelEvent(m.eventsCh)

	case TunnelEventMsg:
		m.appendTraffic(eventToLogLine(UIEvent(msg)))
		return m, waitForTunnelEvent(m.eventsCh)

	case tunnelDoneMsg:
		return m, tea.Quit
	}
	return m, waitForTunnelEvent(m.eventsCh)
}

func eventToLogLine(e UIEvent) string {
	switch e.Type {
	case EventConnOpen:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Render("● Client Connected" + optRemote(e.Remote))
	case EventConnClose:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B")).Render("○ Client Disconnected")
	case EventTraffic:
		if e.Info != nil && e.Info.Protocol == "HTTP" {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#43BF6D")).Render(e.Msg)
		}
		if strings.HasPrefix(e.Msg, "[FAIL]") || strings.HasPrefix(e.Msg, "[Error]") {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#F25D94")).Render(e.Msg)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#874BFD")).Render(e.Msg)
	default:
		return e.Msg
	}
}

func optRemote(r string) string {
	if r == "" {
		return ""
	}
	return " (" + r + ")"
}

func (m *tunnelModel) appendTraffic(line string) {
	m.trafficLog = append(m.trafficLog, line)
	if len(m.trafficLog) > maxTrafficLogs {
		m.trafficLog = m.trafficLog[len(m.trafficLog)-maxTrafficLogs:]
	}
}

// tunnelDoneMsg is sent when the tunnel event channel is closed.
type tunnelDoneMsg struct{}

func waitForTunnelEvent(ch <-chan UIEvent) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		e, ok := <-ch
		if !ok {
			return tunnelDoneMsg{}
		}
		return TunnelEventMsg(e)
	}
}

func (m tunnelModel) View() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(uiHighlight).
		Padding(1, 2).
		Width(68)

	var b strings.Builder

	// Header
	title := "Tunnel Ready (" + m.role + ")"
	b.WriteString(lipgloss.NewStyle().Foreground(uiSpecial).Render(title))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Code: "))
	b.WriteString(lipgloss.NewStyle().Foreground(uiHighlight).Bold(true).Render(m.code))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Addr: "))
	b.WriteString(lipgloss.NewStyle().Foreground(uiHighlight).Render(m.addr))
	b.WriteString("\n\n")

	// Live Traffic panel
	b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("─── Live Traffic ───"))
	b.WriteString("\n")
	if len(m.trafficLog) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("  (waiting for connections...)"))
	} else {
		for _, line := range m.trafficLog {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Press q or Esc to exit"))

	return box.Render(b.String())
}

// RunTunnelUI runs the tunnel with a Bubble Tea TUI. fn should block running the tunnel.
// It creates an event channel, starts fn in a goroutine with opts.Events set,
// and runs the TUI. When the user quits (q/Esc), the process exits.
func RunTunnelUI(role, code, addr string, fn func(opts *TunnelOptions) error) error {
	ch := make(chan UIEvent, 64)
	opts := &TunnelOptions{Events: ch}

	go func() {
		if err := fn(opts); err != nil {
			select {
			case ch <- UIEvent{Type: EventTraffic, Msg: "[Error] " + err.Error()}:
			default:
			}
		}
		close(ch)
	}()

	m := tunnelModel{
		role:     role,
		code:     code,
		addr:     addr,
		eventsCh: ch,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
