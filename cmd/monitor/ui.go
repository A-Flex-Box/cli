package monitor

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the TUI state.
type Model struct {
	opts         *Options
	capture      *Capture
	routeMonitor *RouteMonitor
	conntrackMon *ConnTrackMonitor
	topology     *NetworkTopology
	bandwidthMon *BandwidthMonitor
	dnsMon       *DNSMonitor

	// UI components
	activeTab     int
	packets       []PacketInfo
	routes        []RouteEntry
	connections   []ConnTrackEntry
	dnsQueries    []DNSQuery
	bandwidth     BandwidthStats
	stats         Stats
	help          help.Model
	keys          keyMap
	table         table.Model
	ready         bool
	err           error
	width, height int
	refreshTimer  timer.Model
	lastRefresh   time.Time
}

// Stats holds aggregate statistics.
type Stats struct {
	CaptureStats
	RouteCount   int
	ConnCount    int
	ConnByProto  map[string]int
	ConnByState  map[string]int
	DNSCount     int
	BandwidthBPS int64
	BandwidthPPS int64
}

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Tab     key.Binding
	Help    key.Binding
	Quit    key.Binding
	Refresh key.Binding
	Filter  key.Binding
	Clear   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Left, k.Right, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Tab, k.Refresh, k.Filter, k.Clear},
		{k.Help, k.Quit},
	}
}

var defaultKeyMap = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "prev tab"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "next tab"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	Clear: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear"),
	),
}

// tickMsg is sent periodically to update data.
type tickMsg struct{}

// errMsg represents an error.
type errMsg struct{ error }

// Tab names.
const (
	TabPackets = iota
	TabRoutes
	TabConns
	TabTopology
	TabStats
	TabBandwidth
	TabDNS
	TabCount
)

var tabNames = []string{"Packets", "Routes", "Connections", "Topology", "Stats", "Bandwidth", "DNS"}

// Styles.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#666666"))

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			Underline(true)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(0, 1)
)

// RunMonitorTUI launches the TUI.
func RunMonitorTUI(opts *Options) error {
	if opts.RefreshRate <= 0 {
		opts.RefreshRate = 1
	}

	m := Model{
		opts:         opts,
		keys:         defaultKeyMap,
		help:         help.New(),
		packets:      make([]PacketInfo, 0),
		routes:       make([]RouteEntry, 0),
		connections:  make([]ConnTrackEntry, 0),
		dnsQueries:   make([]DNSQuery, 0),
		refreshTimer: timer.NewWithInterval(time.Duration(opts.RefreshRate)*time.Second, time.Second),
	}

	// Initialize components
	if opts.EnableCapture {
		m.capture = NewCapture(opts.Interface, opts.BPFFilter)
	}
	if opts.EnableRoutes {
		m.routeMonitor = NewRouteMonitor(opts.RefreshRate)
	}
	if opts.EnableConntrack {
		m.conntrackMon = NewConnTrackMonitor(opts.RefreshRate)
	}
	m.topology = NewNetworkTopology()
	m.bandwidthMon = NewBandwidthMonitor(opts.Interface, time.Duration(opts.RefreshRate)*time.Second)
	m.dnsMon = NewDNSMonitor()

	p := tea.NewProgram(&m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshTimer.Init(),
		m.startMonitoring(),
	)
}

func (m *Model) startMonitoring() tea.Cmd {
	return func() tea.Msg {
		// Start capture
		if m.capture != nil {
			if err := m.capture.Start(); err != nil {
				return errMsg{err}
			}
		}

		// Start route monitor
		if m.routeMonitor != nil {
			if err := m.routeMonitor.Start(); err != nil {
				return errMsg{err}
			}
		}

		// Start conntrack monitor
		if m.conntrackMon != nil {
			if err := m.conntrackMon.Start(); err != nil {
				return errMsg{err}
			}
		}

		// Start bandwidth monitor
		if m.bandwidthMon != nil {
			m.bandwidthMon.Start()
		}

		// Start DNS monitor
		if m.dnsMon != nil {
			m.dnsMon.Start()
		}

		m.ready = true
		m.lastRefresh = time.Now()
		return nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keys.Left):
			m.activeTab = (m.activeTab - 1 + TabCount) % TabCount

		case key.Matches(msg, m.keys.Right, m.keys.Tab):
			m.activeTab = (m.activeTab + 1) % TabCount

		case key.Matches(msg, m.keys.Refresh):
			m.refreshData()

		case key.Matches(msg, m.keys.Clear):
			m.clearCurrentTab()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case timer.TickMsg:
		// Refresh data on timer tick
		m.refreshData()

	case timer.TimeoutMsg:
		// Reset timer for next refresh
		m.refreshTimer = timer.NewWithInterval(time.Duration(m.opts.RefreshRate)*time.Second, time.Second)
		cmds = append(cmds, m.refreshTimer.Init())

	case errMsg:
		m.err = msg
	}

	// Update timer
	var timerCmd tea.Cmd
	m.refreshTimer, timerCmd = m.refreshTimer.Update(msg)
	cmds = append(cmds, timerCmd)

	// Update help
	m.help, _ = m.help.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m *Model) refreshData() {
	m.lastRefresh = time.Now()

	// Get capture stats
	if m.capture != nil {
		m.packets = m.capture.GetPackets()
		m.stats.CaptureStats = m.capture.GetStats()
	}

	// Get routes
	if m.routeMonitor != nil {
		m.routes = m.routeMonitor.GetRoutes()
		m.stats.RouteCount = len(m.routes)
	}

	// Get connections
	if m.conntrackMon != nil {
		m.connections = m.conntrackMon.GetConnections()
		m.stats.ConnCount = len(m.connections)
		connStats := m.conntrackMon.GetStats()
		m.stats.ConnByProto = connStats.ByProto
		m.stats.ConnByState = connStats.ByState
	}

	// Get bandwidth stats
	if m.bandwidthMon != nil {
		m.bandwidth = m.bandwidthMon.GetStats()
		m.stats.BandwidthBPS = m.bandwidth.BytesPerSecond
		m.stats.BandwidthPPS = m.bandwidth.PacketsPerSecond
	}

	// Get DNS stats
	if m.dnsMon != nil {
		m.dnsQueries = m.dnsMon.GetQueries()
		m.stats.DNSCount = len(m.dnsQueries)
	}

	// Update topology
	if m.topology != nil {
		if m.capture != nil {
			m.topology.UpdateFromPackets(m.packets)
		}
		if m.conntrackMon != nil {
			m.topology.UpdateFromConnections(m.connections)
		}
		if m.routeMonitor != nil {
			if _, gw := m.routeMonitor.GetDefaultGateway(); gw != "" {
				m.topology.SetGateway(gw)
			}
		}
	}
}

func (m *Model) clearCurrentTab() {
	switch m.activeTab {
	case TabPackets:
		m.packets = make([]PacketInfo, 0)
	case TabTopology:
		if m.topology != nil {
			m.topology.Clear()
		}
	}
}

func (m *Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	var sections []string

	// Title
	title := titleStyle.Render("🔍 Network Monitor")
	sections = append(sections, title)

	// Tabs
	tabs := m.renderTabs()
	sections = append(sections, tabs)

	// Content
	content := m.renderContent()
	sections = append(sections, content)

	// Status bar
	status := m.renderStatus()
	sections = append(sections, status)

	// Help
	help := m.help.View(m.keys)
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *Model) renderTabs() string {
	var tabs []string
	for i, name := range tabNames {
		style := tabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		tabs = append(tabs, style.Render(name))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m *Model) renderContent() string {
	contentHeight := m.height - 10 // Account for header, tabs, status, help
	if contentHeight < 10 {
		contentHeight = 10
	}

	switch m.activeTab {
	case TabPackets:
		return m.renderPackets(contentHeight)
	case TabRoutes:
		return m.renderRoutes(contentHeight)
	case TabConns:
		return m.renderConnections(contentHeight)
	case TabTopology:
		return m.renderTopology(contentHeight)
	case TabStats:
		return m.renderStats(contentHeight)
	case TabBandwidth:
		return m.renderBandwidth(contentHeight)
	case TabDNS:
		return m.renderDNS(contentHeight)
	default:
		return ""
	}
}

func (m *Model) renderPackets(height int) string {
	if len(m.packets) == 0 {
		if m.capture == nil {
			return boxStyle.Render("Packet capture disabled")
		}
		return boxStyle.Render("No packets captured yet...")
	}

	rows := make([]string, 0, height)
	header := headerStyle.Render(
		fmt.Sprintf("%-22s %-22s %-8s %-8s %s",
			"Source", "Destination", "Protocol", "Size", "Time"))
	rows = append(rows, header)

	// Show most recent packets first
	start := len(m.packets) - height + 1
	if start < 0 {
		start = 0
	}

	for i := start; i < len(m.packets); i++ {
		p := m.packets[i]
		timeStr := p.Timestamp.Format("15:04:05.000")
		row := fmt.Sprintf("%-22s %-22s %-8s %-8d %s",
			fmt.Sprintf("%s:%d", p.SrcIP, p.SrcPort),
			fmt.Sprintf("%s:%d", p.DstIP, p.DstPort),
			p.Protocol,
			p.Length,
			timeStr,
		)
		rows = append(rows, itemStyle.Render(row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *Model) renderRoutes(height int) string {
	if len(m.routes) == 0 {
		if m.routeMonitor == nil {
			return boxStyle.Render("Route monitoring disabled")
		}
		return boxStyle.Render("No routes found...")
	}

	rows := make([]string, 0, height)
	header := headerStyle.Render(
		fmt.Sprintf("%-18s %-18s %-18s %-10s %s",
			"Destination", "Gateway", "Genmask", "Flags", "Interface"))
	rows = append(rows, header)

	for i, r := range m.routes {
		if i >= height-1 {
			break
		}
		row := fmt.Sprintf("%-18s %-18s %-18s %-10s %s",
			r.Destination,
			r.Gateway,
			r.Genmask,
			r.Flags,
			r.Interface,
		)
		rows = append(rows, itemStyle.Render(row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *Model) renderConnections(height int) string {
	if len(m.connections) == 0 {
		if m.conntrackMon == nil {
			return boxStyle.Render("Connection tracking disabled")
		}
		return boxStyle.Render("No connections tracked...")
	}

	rows := make([]string, 0, height)
	header := headerStyle.Render(
		fmt.Sprintf("%-6s %-22s %-22s %-12s %s",
			"Proto", "Source", "Destination", "State", "Bytes"))
	rows = append(rows, header)

	for i, c := range m.connections {
		if i >= height-1 {
			break
		}
		totalBytes := c.BytesSrc + c.BytesDst
		row := fmt.Sprintf("%-6s %-22s %-22s %-12s %s",
			c.Proto,
			fmt.Sprintf("%s:%d", c.SrcIP, c.SrcPort),
			fmt.Sprintf("%s:%d", c.DstIP, c.DstPort),
			c.State,
			formatBytes(totalBytes),
		)
		rows = append(rows, itemStyle.Render(row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *Model) renderTopology(height int) string {
	if m.topology == nil {
		return boxStyle.Render("Topology visualization disabled")
	}

	nodes := m.topology.GetNodes()
	if len(nodes) == 0 {
		return boxStyle.Render("No network topology data available. Waiting for traffic...")
	}

	return m.topology.GetTopologyGraph()
}

func (m *Model) renderStats(height int) string {
	var sections []string

	// Capture stats
	if m.capture != nil {
		stats := m.capture.GetStats()
		captureStats := []string{
			headerStyle.Render("Packet Capture"),
			fmt.Sprintf("  Total Packets: %d", stats.TotalPackets),
			fmt.Sprintf("  Total Bytes:   %s", formatBytes(stats.TotalBytes)),
			fmt.Sprintf("  By Protocol:"),
		}
		for proto, count := range stats.ByProtocol {
			captureStats = append(captureStats, fmt.Sprintf("    %s: %d", proto, count))
		}
		sections = append(sections, boxStyle.Render(strings.Join(captureStats, "\n")))
	}

	// Route stats
	if m.routeMonitor != nil {
		routeStats := []string{
			headerStyle.Render("Routing Table"),
			fmt.Sprintf("  Total Routes: %d", m.stats.RouteCount),
		}
		if iface, gw := m.routeMonitor.GetDefaultGateway(); gw != "" {
			routeStats = append(routeStats, fmt.Sprintf("  Default GW:   %s (%s)", gw, iface))
		}
		sections = append(sections, boxStyle.Render(strings.Join(routeStats, "\n")))
	}

	// Connection stats
	if m.conntrackMon != nil {
		connStats := m.conntrackMon.GetStats()
		connStatsDisplay := []string{
			headerStyle.Render("Connection Tracking"),
			fmt.Sprintf("  Total: %d", connStats.Total),
			fmt.Sprintf("  By Protocol:"),
		}
		for proto, count := range connStats.ByProto {
			connStatsDisplay = append(connStatsDisplay, fmt.Sprintf("    %s: %d", proto, count))
		}
		connStatsDisplay = append(connStatsDisplay, "  By State:")
		for state, count := range connStats.ByState {
			connStatsDisplay = append(connStatsDisplay, fmt.Sprintf("    %s: %d", state, count))
		}
		sections = append(sections, boxStyle.Render(strings.Join(connStatsDisplay, "\n")))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *Model) renderBandwidth(height int) string {
	if m.bandwidthMon == nil {
		return boxStyle.Render("Bandwidth monitoring disabled")
	}

	stats := m.bandwidth

	var sections []string
	sections = append(sections, headerStyle.Render("Bandwidth Statistics"))
	sections = append(sections, "")
	sections = append(sections, fmt.Sprintf("  Current:  %s / %s", FormatBandwidth(stats.BytesPerSecond), stats.PacketsPerSecond)+" pps")
	sections = append(sections, fmt.Sprintf("  Peak:     %s / %d pps", FormatBandwidth(stats.PeakBPS), stats.PeakPPS))
	sections = append(sections, fmt.Sprintf("  Average:  %s / %d pps", FormatBandwidth(stats.AvgBPS), stats.AvgPPS))
	sections = append(sections, "")
	sections = append(sections, fmt.Sprintf("  In:       %s", FormatBandwidth(stats.BytesPerSecondIn)))
	sections = append(sections, fmt.Sprintf("  Out:      %s", FormatBandwidth(stats.BytesPerSecondOut)))
	sections = append(sections, "")
	sections = append(sections, fmt.Sprintf("  Total:    %s / %d packets", formatBytes(stats.TotalBytes), stats.TotalPackets))

	history := m.bandwidthMon.GetHistory()
	if len(history) > 0 {
		sections = append(sections, "")
		sections = append(sections, headerStyle.Render("Recent History (last 10 samples)"))

		samples := history
		if len(samples) > 10 {
			samples = samples[len(samples)-10:]
		}

		for _, s := range samples {
			sections = append(sections, fmt.Sprintf("  %s: %s / %d pps",
				s.Timestamp.Format("15:04:05"),
				FormatBandwidth(s.BytesPerSec),
				s.PacketsPerSec))
		}
	}

	return boxStyle.Render(strings.Join(sections, "\n"))
}

func (m *Model) renderDNS(height int) string {
	if m.dnsMon == nil {
		return boxStyle.Render("DNS monitoring disabled")
	}

	var sections []string

	stats := m.dnsMon.GetStats()
	sections = append(sections, headerStyle.Render("DNS Statistics"))
	sections = append(sections, "")
	sections = append(sections, fmt.Sprintf("  Total Queries:  %d", stats.TotalQueries))
	sections = append(sections, fmt.Sprintf("  Unique Domains: %d", stats.UniqueDomains))
	sections = append(sections, fmt.Sprintf("  Queries/sec:    %.1f", stats.QueriesPerSec))
	sections = append(sections, fmt.Sprintf("  Pending:        %d", stats.PendingQueries))
	if stats.AvgLatency > 0 {
		sections = append(sections, fmt.Sprintf("  Avg Latency:    %v", stats.AvgLatency.Round(time.Millisecond)))
	}

	topDomains := m.dnsMon.GetTopDomains(10)
	if len(topDomains) > 0 {
		sections = append(sections, "")
		sections = append(sections, headerStyle.Render("Top Domains"))
		for _, d := range topDomains {
			sections = append(sections, fmt.Sprintf("  %-30s %d", d.Domain, d.Count))
		}
	}

	queries := m.dnsQueries
	if len(queries) > 0 {
		sections = append(sections, "")
		sections = append(sections, headerStyle.Render("Recent Queries"))

		start := len(queries) - 10
		if start < 0 {
			start = 0
		}

		for i := start; i < len(queries); i++ {
			q := queries[i]
			domain := q.Domain
			if domain == "" {
				domain = fmt.Sprintf("%s:%d", q.DstIP, q.DstPort)
			}
			sections = append(sections, fmt.Sprintf("  %s %s (%s)",
				q.Timestamp.Format("15:04:05.000"),
				domain,
				q.QueryType))
		}
	}

	return boxStyle.Render(strings.Join(sections, "\n"))
}

func (m *Model) renderStatus() string {
	var parts []string

	// Interface info
	if m.capture != nil && m.capture.iface != "" {
		parts = append(parts, fmt.Sprintf("Interface: %s", m.capture.iface))
	}

	// Refresh time
	parts = append(parts, fmt.Sprintf("Updated: %s", m.lastRefresh.Format("15:04:05")))

	// Counts
	parts = append(parts, fmt.Sprintf("Packets: %d", len(m.packets)))
	parts = append(parts, fmt.Sprintf("Routes: %d", len(m.routes)))
	parts = append(parts, fmt.Sprintf("Conns: %d", len(m.connections)))

	return statusStyle.Render(strings.Join(parts, " | "))
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
