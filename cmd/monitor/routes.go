package monitor

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

// RouteEntry represents a single routing table entry.
type RouteEntry struct {
	Destination string
	Gateway     string
	Genmask     string
	Flags       string
	Metric      int
	Interface   string
	Mtu         int
	Window      int
	IRTT        int
}

// RouteMonitor monitors the system routing table.
type RouteMonitor struct {
	mu          sync.RWMutex
	routes      []RouteEntry
	events      chan RouteEvent
	stopChan    chan struct{}
	running     bool
	refreshRate time.Duration
}

// RouteEvent represents a route change event.
type RouteEvent struct {
	Type    string     // "add", "del", "change"
	Route   RouteEntry
	When    time.Time
}

// NewRouteMonitor creates a new route monitor.
func NewRouteMonitor(refreshSeconds int) *RouteMonitor {
	return &RouteMonitor{
		routes:      make([]RouteEntry, 0),
		events:      make(chan RouteEvent, 50),
		stopChan:    make(chan struct{}),
		refreshRate: time.Duration(refreshSeconds) * time.Second,
	}
}

// Start begins monitoring the routing table.
func (m *RouteMonitor) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	// Initial load
	routes, err := m.loadRoutes()
	if err != nil {
		logger.Debug("initial route load failed", zap.Error(err))
	}
	m.mu.Lock()
	m.routes = routes
	m.mu.Unlock()

	go m.monitorLoop()

	logger.Debug("route monitor started")
	return nil
}

// Stop halts the route monitor.
func (m *RouteMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopChan)
	m.running = false

	logger.Debug("route monitor stopped")
}

// Events returns the route event channel.
func (m *RouteMonitor) Events() <-chan RouteEvent {
	return m.events
}

// GetRoutes returns current routing table.
func (m *RouteMonitor) GetRoutes() []RouteEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]RouteEntry, len(m.routes))
	copy(result, m.routes)
	return result
}

func (m *RouteMonitor) monitorLoop() {
	ticker := time.NewTicker(m.refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkRoutes()
		}
	}
}

func (m *RouteMonitor) checkRoutes() {
	newRoutes, err := m.loadRoutes()
	if err != nil {
		return
	}

	m.mu.RLock()
	oldRoutes := m.routes
	m.mu.RUnlock()

	// Compare routes and detect changes
	routeMap := make(map[string]RouteEntry)
	for _, r := range oldRoutes {
		key := fmt.Sprintf("%s:%s:%s", r.Destination, r.Gateway, r.Interface)
		routeMap[key] = r
	}

	for _, r := range newRoutes {
		key := fmt.Sprintf("%s:%s:%s", r.Destination, r.Gateway, r.Interface)
		if _, exists := routeMap[key]; !exists {
			// New route added
			select {
			case m.events <- RouteEvent{Type: "add", Route: r, When: time.Now()}:
			default:
			}
		}
		delete(routeMap, key)
	}

	// Remaining routes were deleted
	for _, r := range routeMap {
		select {
		case m.events <- RouteEvent{Type: "del", Route: r, When: time.Now()}:
		default:
		}
	}

	m.mu.Lock()
	m.routes = newRoutes
	m.mu.Unlock()
}

// loadRoutes reads the routing table from /proc/net/route or netlink.
func (m *RouteMonitor) loadRoutes() ([]RouteEntry, error) {
	// First try /proc/net/route (Linux)
	if _, err := os.Stat("/proc/net/route"); err == nil {
		return m.loadRoutesFromProc()
	}

	// Fallback: try ip route command
	return m.loadRoutesFromIPCommand()
}

func (m *RouteMonitor) loadRoutesFromProc() ([]RouteEntry, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var routes []RouteEntry
	scanner := bufio.NewScanner(file)

	// Skip header
	scanner.Scan()

	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		if len(line) < 8 {
			continue
		}

		dest := line[1]
		gateway := line[2]
		flags := line[3]
		genmask := line[7]
		iface := line[0]

		metric := 0
		if len(line) > 4 {
			fmt.Sscanf(line[4], "%d", &metric)
		}

		mtu := 0
		if len(line) > 6 {
			fmt.Sscanf(line[6], "%d", &mtu)
		}

		route := RouteEntry{
			Destination: formatHexIP(dest),
			Gateway:     formatHexIP(gateway),
			Genmask:     formatHexIP(genmask),
			Flags:       parseRouteFlags(flags),
			Metric:      metric,
			Interface:   iface,
			Mtu:         mtu,
		}

		routes = append(routes, route)
	}

	return routes, scanner.Err()
}

func (m *RouteMonitor) loadRoutesFromIPCommand() ([]RouteEntry, error) {
	// Simple fallback - would need proper implementation for non-Linux
	// or when /proc is not available
	return nil, fmt.Errorf("ip command fallback not implemented")
}

func formatHexIP(hex string) string {
	if hex == "00000000" {
		return "0.0.0.0"
	}

	// Convert hex to IP (reversed byte order in /proc/net/route)
	if len(hex) != 8 {
		return hex
	}

	ip := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		var b byte
		fmt.Sscanf(hex[6-2*i:8-2*i], "%02x", &b)
		ip[i] = b
	}

	return ip.String()
}

func parseRouteFlags(flags string) string {
	result := ""
	flagMap := map[rune]string{
		'U': "UP",
		'G': "GATEWAY",
		'H': "HOST",
		'R': "REINSTATE",
		'D': "DYNAMIC",
		'M': "MODIFIED",
		'A': "ADDRCONF",
		'C': "CACHE",
		'!': "REJECT",
	}

	for i, c := range flags {
		if f, ok := flagMap[c]; ok {
			if i > 0 {
				result += ","
			}
			result += f
		}
	}

	if result == "" {
		return flags
	}
	return result
}

// GetRoutingTable returns a formatted string of routes (utility function).
func GetRoutingTable() (string, error) {
	// Check if route command exists
	if _, err := os.Stat("/proc/net/route"); err == nil {
		data, err := os.ReadFile("/proc/net/route")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	return "", fmt.Errorf("cannot read route table")
}

// FindFile finds a file in standard Linux proc locations.
func FindFile(name string) string {
	locations := []string{
		"/proc/net/" + name,
		"/proc/" + name,
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	return ""
}

// ParseCIDR converts an IP with netmask to CIDR notation.
func ParseCIDR(ip, mask string) string {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return ""
	}

	maskIP := net.ParseIP(mask)
	if maskIP == nil {
		// Try parsing as integer mask
		var ones int
		fmt.Sscanf(mask, "%d", &ones)
		_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", ip, ones))
		if err != nil {
			return ""
		}
		return ipNet.String()
	}

	ones, _ := net.IPMask(maskIP.To4()).Size()
	_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", ip, ones))
	if err != nil {
		return ""
	}
	return ipNet.String()
}

// GetDefaultGateway returns the default gateway interface and IP.
func (m *RouteMonitor) GetDefaultGateway() (string, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, r := range m.routes {
		if r.Destination == "0.0.0.0" && strings.Contains(r.Flags, "GATEWAY") {
			return r.Interface, r.Gateway
		}
	}

	return "", ""
}

// RouteTablePath returns the path to the route table.
var RouteTablePath = filepath.Join("proc", "net", "route")