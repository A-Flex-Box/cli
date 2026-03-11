package monitor

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

type BandwidthStats struct {
	BytesPerSecond    int64
	PacketsPerSecond  int64
	BytesPerSecondIn  int64
	BytesPerSecondOut int64
	PeakBPS           int64
	PeakPPS           int64
	AvgBPS            int64
	AvgPPS            int64
	TotalBytes        int64
	TotalPackets      int64
	StartTime         time.Time
}

type BandwidthMonitor struct {
	mu         sync.RWMutex
	iface      string
	sampleRate time.Duration
	stopChan   chan struct{}
	running    bool

	lastBytes    int64
	lastPackets  int64
	lastBytesIn  int64
	lastBytesOut int64
	lastSample   time.Time

	current    BandwidthStats
	history    []BandwidthSample
	maxHistory int
}

type BandwidthSample struct {
	Timestamp     time.Time
	BytesPerSec   int64
	PacketsPerSec int64
}

func NewBandwidthMonitor(iface string, sampleRate time.Duration) *BandwidthMonitor {
	if sampleRate <= 0 {
		sampleRate = time.Second
	}
	return &BandwidthMonitor{
		iface:      iface,
		sampleRate: sampleRate,
		stopChan:   make(chan struct{}),
		maxHistory: 3600,
		history:    make([]BandwidthSample, 0, 100),
	}
}

func (m *BandwidthMonitor) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.current.StartTime = time.Now()
	m.lastSample = time.Now()
	m.mu.Unlock()

	go m.monitorLoop()

	logger.Debug("bandwidth monitor started", zap.String("interface", m.iface))
	return nil
}

func (m *BandwidthMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopChan)
	m.running = false

	logger.Debug("bandwidth monitor stopped")
}

func (m *BandwidthMonitor) GetStats() BandwidthStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *BandwidthMonitor) GetHistory() []BandwidthSample {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]BandwidthSample, len(m.history))
	copy(result, m.history)
	return result
}

func (m *BandwidthMonitor) monitorLoop() {
	ticker := time.NewTicker(m.sampleRate)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.sample()
		}
	}
}

func (m *BandwidthMonitor) sample() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastSample).Seconds()
	if elapsed <= 0 {
		return
	}

	stats, err := getInterfaceStats(m.iface)
	if err != nil {
		return
	}

	bytesDiff := stats.RxBytes + stats.TxBytes - m.lastBytes - m.lastBytesOut
	packetsDiff := stats.RxPackets + stats.TxPackets - m.lastPackets

	bps := int64(float64(bytesDiff) / elapsed)
	pps := int64(float64(packetsDiff) / elapsed)

	m.current.BytesPerSecond = bps
	m.current.PacketsPerSecond = pps
	m.current.BytesPerSecondIn = int64(float64(stats.RxBytes-m.lastBytesIn) / elapsed)
	m.current.BytesPerSecondOut = int64(float64(stats.TxBytes-m.lastBytesOut) / elapsed)
	m.current.TotalBytes = stats.RxBytes + stats.TxBytes
	m.current.TotalPackets = stats.RxPackets + stats.TxPackets

	if bps > m.current.PeakBPS {
		m.current.PeakBPS = bps
	}
	if pps > m.current.PeakPPS {
		m.current.PeakPPS = pps
	}

	totalSamples := len(m.history) + 1
	m.current.AvgBPS = (m.current.AvgBPS*int64(len(m.history)) + bps) / int64(totalSamples)
	m.current.AvgPPS = (m.current.AvgPPS*int64(len(m.history)) + pps) / int64(totalSamples)

	m.history = append(m.history, BandwidthSample{
		Timestamp:     now,
		BytesPerSec:   bps,
		PacketsPerSec: pps,
	})

	if len(m.history) > m.maxHistory {
		m.history = m.history[len(m.history)-m.maxHistory:]
	}

	m.lastBytes = stats.RxBytes + stats.TxBytes
	m.lastBytesIn = stats.RxBytes
	m.lastBytesOut = stats.TxBytes
	m.lastPackets = stats.RxPackets + stats.TxPackets
	m.lastSample = now
}

func (m *BandwidthMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = BandwidthStats{StartTime: time.Now()}
	m.history = make([]BandwidthSample, 0, 100)
	m.lastBytes = 0
	m.lastPackets = 0
	m.lastBytesIn = 0
	m.lastBytesOut = 0
	m.lastSample = time.Now()
}

func (m *BandwidthMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

type InterfaceStats struct {
	RxBytes   int64
	RxPackets int64
	RxErrors  int64
	RxDropped int64
	TxBytes   int64
	TxPackets int64
	TxErrors  int64
	TxDropped int64
}

func getInterfaceStats(iface string) (InterfaceStats, error) {
	if iface == "" {
		var err error
		iface, err = GetDefaultInterfaceName()
		if err != nil {
			return InterfaceStats{}, err
		}
	}

	return readInterfaceStatsFromProc(iface)
}

func GetDefaultInterfaceName() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.IsGlobalUnicast() {
				return iface.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable interface found")
}

func GetAvailableInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 {
			result = append(result, iface)
		}
	}
	return result, nil
}

func readInterfaceStatsFromProc(iface string) (InterfaceStats, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return InterfaceStats{}, fmt.Errorf("failed to read /proc/net/dev: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if !strings.Contains(line, iface+":") {
			continue
		}

		parts := strings.Fields(strings.TrimPrefix(line, iface+":"))
		if len(parts) < 16 {
			continue
		}

		return InterfaceStats{
			RxBytes:   parseInt64(parts[0]),
			RxPackets: parseInt64(parts[1]),
			RxErrors:  parseInt64(parts[2]),
			RxDropped: parseInt64(parts[3]),
			TxBytes:   parseInt64(parts[8]),
			TxPackets: parseInt64(parts[9]),
			TxErrors:  parseInt64(parts[10]),
			TxDropped: parseInt64(parts[11]),
		}, nil
	}

	return InterfaceStats{}, fmt.Errorf("interface %s not found in /proc/net/dev", iface)
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

type BandwidthWatcher struct {
	monitors map[string]*BandwidthMonitor
	mu       sync.RWMutex
	stopChan chan struct{}
	running  atomic.Bool
}

func NewBandwidthWatcher() *BandwidthWatcher {
	return &BandwidthWatcher{
		monitors: make(map[string]*BandwidthMonitor),
		stopChan: make(chan struct{}),
	}
}

func (w *BandwidthWatcher) AddInterface(iface string, sampleRate time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.monitors[iface]; !exists {
		w.monitors[iface] = NewBandwidthMonitor(iface, sampleRate)
		if w.running.Load() {
			w.monitors[iface].Start()
		}
	}
}

func (w *BandwidthWatcher) RemoveInterface(iface string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if mon, exists := w.monitors[iface]; exists {
		mon.Stop()
		delete(w.monitors, iface)
	}
}

func (w *BandwidthWatcher) Start() {
	if w.running.Swap(true) {
		return
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, mon := range w.monitors {
		mon.Start()
	}
}

func (w *BandwidthWatcher) Stop() {
	if !w.running.Swap(false) {
		return
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, mon := range w.monitors {
		mon.Stop()
	}
}

func (w *BandwidthWatcher) GetStats(iface string) (BandwidthStats, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if mon, exists := w.monitors[iface]; exists {
		return mon.GetStats(), true
	}
	return BandwidthStats{}, false
}

func (w *BandwidthWatcher) GetAllStats() map[string]BandwidthStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string]BandwidthStats)
	for name, mon := range w.monitors {
		result[name] = mon.GetStats()
	}
	return result
}

func FormatBandwidth(bps int64) string {
	const unit = 1024
	if bps < unit {
		return fmt.Sprintf("%d B/s", bps)
	}

	div, exp := int64(unit), 0
	for n := bps / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB/s", float64(bps)/float64(div), "KMGTPE"[exp])
}
