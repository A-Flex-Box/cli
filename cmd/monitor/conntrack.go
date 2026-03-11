package monitor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

// ConnTrackEntry represents a connection tracking entry.
type ConnTrackEntry struct {
	Proto       string
	SrcIP       string
	SrcPort     int
	DstIP       string
	DstPort     int
	State       string
	Timeout     int
	BytesSrc    int64
	BytesDst    int64
	PacketsSrc  int64
	PacketsDst  int64
	Mark        string
	Zone        int
	StartTime   time.Time
}

// ConnTrackMonitor monitors connection tracking table.
type ConnTrackMonitor struct {
	mu           sync.RWMutex
	connections  []ConnTrackEntry
	events       chan ConnTrackEvent
	stopChan     chan struct{}
	running      bool
	refreshRate  time.Duration
	useConntrack bool // true if conntrack tool is available
}

// ConnTrackEvent represents a connection state change.
type ConnTrackEvent struct {
	Type      string         // "new", "update", "destroy"
	Entry     ConnTrackEntry
	When      time.Time
}

// ConnTrackStats aggregates connection statistics.
type ConnTrackStats struct {
	Total      int
	ByProto    map[string]int
	ByState    map[string]int
	TotalBytes int64
}

// NewConnTrackMonitor creates a new connection tracking monitor.
func NewConnTrackMonitor(refreshSeconds int) *ConnTrackMonitor {
	m := &ConnTrackMonitor{
		connections: make([]ConnTrackEntry, 0),
		events:      make(chan ConnTrackEvent, 50),
		stopChan:    make(chan struct{}),
		refreshRate: time.Duration(refreshSeconds) * time.Second,
	}

	// Check if conntrack tool exists
	if _, err := exec.LookPath("conntrack"); err == nil {
		m.useConntrack = true
	}

	return m
}

// Start begins monitoring connections.
func (m *ConnTrackMonitor) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	// Initial load
	conns, err := m.loadConnectionTable()
	if err != nil {
		logger.Debug("initial conntrack load failed", zap.Error(err))
	}
	m.mu.Lock()
	m.connections = conns
	m.mu.Unlock()

	go m.monitorLoop()

	logger.Debug("conntrack monitor started", zap.Bool("conntrack_tool", m.useConntrack))
	return nil
}

// Stop halts the connection monitor.
func (m *ConnTrackMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopChan)
	m.running = false

	logger.Debug("conntrack monitor stopped")
}

// Events returns the connection event channel.
func (m *ConnTrackMonitor) Events() <-chan ConnTrackEvent {
	return m.events
}

// GetConnections returns current connections.
func (m *ConnTrackMonitor) GetConnections() []ConnTrackEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ConnTrackEntry, len(m.connections))
	copy(result, m.connections)
	return result
}

// GetStats returns aggregated connection statistics.
func (m *ConnTrackMonitor) GetStats() ConnTrackStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := ConnTrackStats{
		Total:    len(m.connections),
		ByProto:  make(map[string]int),
		ByState:  make(map[string]int),
	}

	for _, c := range m.connections {
		stats.ByProto[c.Proto]++
		stats.ByState[c.State]++
		stats.TotalBytes += c.BytesSrc + c.BytesDst
	}

	return stats
}

func (m *ConnTrackMonitor) monitorLoop() {
	ticker := time.NewTicker(m.refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkConnections()
		}
	}
}

func (m *ConnTrackMonitor) checkConnections() {
	newConns, err := m.loadConnectionTable()
	if err != nil {
		return
	}

	m.mu.RLock()
	oldConns := m.connections
	m.mu.RUnlock()

	// Build map of old connections
	connMap := make(map[string]ConnTrackEntry)
	for _, c := range oldConns {
		key := connKey(c)
		connMap[key] = c
	}

	// Detect changes
	for _, c := range newConns {
		key := connKey(c)
		if _, exists := connMap[key]; !exists {
			// New connection
			select {
			case m.events <- ConnTrackEvent{Type: "new", Entry: c, When: time.Now()}:
			default:
			}
		}
		delete(connMap, key)
	}

	// Remaining are destroyed
	for _, c := range connMap {
		select {
		case m.events <- ConnTrackEvent{Type: "destroy", Entry: c, When: time.Now()}:
		default:
		}
	}

	m.mu.Lock()
	m.connections = newConns
	m.mu.Unlock()
}

func connKey(c ConnTrackEntry) string {
	return fmt.Sprintf("%s:%d->%s:%d", c.SrcIP, c.SrcPort, c.DstIP, c.DstPort)
}

// loadConnectionTable loads the connection tracking table.
func (m *ConnTrackMonitor) loadConnectionTable() ([]ConnTrackEntry, error) {
	// Prefer conntrack tool if available
	if m.useConntrack {
		return m.loadFromConntrackTool()
	}

	// Fallback to /proc/net/nf_conntrack
	return m.loadFromProc()
}

func (m *ConnTrackMonitor) loadFromConntrackTool() ([]ConnTrackEntry, error) {
	cmd := exec.Command("conntrack", "-L")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []ConnTrackEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		entry := m.parseConntrackLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries, scanner.Err()
}

func (m *ConnTrackMonitor) loadFromProc() ([]ConnTrackEntry, error) {
	file, err := os.Open("/proc/net/nf_conntrack")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []ConnTrackEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		entry := m.parseProcLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries, scanner.Err()
}

func (m *ConnTrackMonitor) parseConntrackLine(line string) *ConnTrackEntry {
	// Parse conntrack -L output format:
	// tcp      6 431942 ESTABLISHED src=192.168.1.100 dst=93.184.216.34 sport=54321 dport=443 src=93.184.216.34 dst=192.168.1.100 sport=443 dport=54321 [ASSURED] mark=0 zone=0
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return nil
	}

	entry := &ConnTrackEntry{
		Proto: fields[0],
	}

	// Parse state
	for _, f := range fields {
		switch {
		case strings.HasPrefix(f, "src="):
			// Only parse the first src (original direction)
			if entry.SrcIP == "" {
				entry.SrcIP = strings.TrimPrefix(f, "src=")
			}
		case strings.HasPrefix(f, "dst="):
			if entry.DstIP == "" {
				entry.DstIP = strings.TrimPrefix(f, "dst=")
			}
		case strings.HasPrefix(f, "sport="):
			if entry.SrcPort == 0 {
				entry.SrcPort, _ = strconv.Atoi(strings.TrimPrefix(f, "sport="))
			}
		case strings.HasPrefix(f, "dport="):
			if entry.DstPort == 0 {
				entry.DstPort, _ = strconv.Atoi(strings.TrimPrefix(f, "dport="))
			}
		case strings.HasPrefix(f, "bytes="):
			bytes, _ := strconv.ParseInt(strings.TrimPrefix(f, "bytes="), 10, 64)
			entry.BytesSrc += bytes
		case strings.HasPrefix(f, "packets="):
			pkts, _ := strconv.ParseInt(strings.TrimPrefix(f, "packets="), 10, 64)
			entry.PacketsSrc += pkts
		case f == "ESTABLISHED" || f == "TIME_WAIT" || f == "CLOSE_WAIT" || f == "SYN_SENT" || f == "SYN_RECV" || f == "FIN_WAIT" || f == "LAST_ACK" || f == "CLOSED":
			if entry.State == "" {
				entry.State = f
			}
		}
	}

	if entry.SrcIP == "" || entry.DstIP == "" {
		return nil
	}

	return entry
}

func (m *ConnTrackMonitor) parseProcLine(line string) *ConnTrackEntry {
	// Parse /proc/net/nf_conntrack format:
	// ipv4     2 tcp      6 431942 ESTABLISHED src=192.168.1.100 dst=93.184.216.34 sport=54321 dport=443 src=93.184.216.34 dst=192.168.1.100 sport=443 dport=54321 [ASSURED] mark=0 zone=0

	entry := &ConnTrackEntry{}
	fields := strings.Fields(line)

	for i, f := range fields {
		if i == 2 {
			entry.Proto = f
			continue
		}

		switch {
		case strings.HasPrefix(f, "src="):
			if entry.SrcIP == "" {
				entry.SrcIP = strings.TrimPrefix(f, "src=")
			}
		case strings.HasPrefix(f, "dst="):
			if entry.DstIP == "" {
				entry.DstIP = strings.TrimPrefix(f, "dst=")
			}
		case strings.HasPrefix(f, "sport="):
			if entry.SrcPort == 0 {
				entry.SrcPort, _ = strconv.Atoi(strings.TrimPrefix(f, "sport="))
			}
		case strings.HasPrefix(f, "dport="):
			if entry.DstPort == 0 {
				entry.DstPort, _ = strconv.Atoi(strings.TrimPrefix(f, "dport="))
			}
		case strings.HasPrefix(f, "bytes="):
			bytes, _ := strconv.ParseInt(strings.TrimPrefix(f, "bytes="), 10, 64)
			entry.BytesSrc += bytes
		case f == "ESTABLISHED" || f == "TIME_WAIT" || f == "CLOSE_WAIT" || f == "SYN_SENT" || f == "SYN_RECV" || f == "FIN_WAIT" || f == "LAST_ACK" || f == "CLOSED":
			if entry.State == "" {
				entry.State = f
			}
		}
	}

	if entry.SrcIP == "" || entry.DstIP == "" {
		return nil
	}

	return entry
}

// IsAvailable returns true if connection tracking is available on this system.
func (m *ConnTrackMonitor) IsAvailable() bool {
	// Check if we can read conntrack
	if _, err := os.Stat("/proc/net/nf_conntrack"); err == nil {
		return true
	}
	if _, err := exec.LookPath("conntrack"); err == nil {
		return true
	}
	return false
}

// FilterByState returns connections matching a state.
func (m *ConnTrackMonitor) FilterByState(state string) []ConnTrackEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ConnTrackEntry
	for _, c := range m.connections {
		if c.State == state {
			result = append(result, c)
		}
	}
	return result
}

// FilterByProto returns connections matching a protocol.
func (m *ConnTrackMonitor) FilterByProto(proto string) []ConnTrackEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ConnTrackEntry
	for _, c := range m.connections {
		if c.Proto == proto {
			result = append(result, c)
		}
	}
	return result
}