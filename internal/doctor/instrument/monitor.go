package instrument

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

// TrafficMonitor wraps an io.ReadWriter to count bytes and calculate real-time speed.
// Implements the middleware pattern for future traffic tracing in Wormhole.
type TrafficMonitor struct {
	rw            io.ReadWriter
	readBytes     atomic.Int64
	writeBytes    atomic.Int64
	speedInterval time.Duration
	onDataHook    func([]byte)
	mu            sync.RWMutex
	stopCh        chan struct{}
}

// OnDataHook is a function type called when data passes through the monitor.
type OnDataHook func(b []byte)

// TrafficMonitorConfig holds configuration for TrafficMonitor.
type TrafficMonitorConfig struct {
	RW            io.ReadWriter
	SpeedInterval time.Duration // e.g. 1s for speed calculation
	OnData        OnDataHook    // optional, feeds data to Sniffer
}

// NewTrafficMonitor creates a TrafficMonitor wrapping the given io.ReadWriter.
func NewTrafficMonitor(cfg TrafficMonitorConfig) *TrafficMonitor {
	if cfg.RW == nil {
		logger.Error("TrafficMonitor: nil io.ReadWriter",
			zap.String("component", "instrument.TrafficMonitor"))
		return nil
	}
	interval := cfg.SpeedInterval
	if interval <= 0 {
		interval = time.Second
	}
	m := &TrafficMonitor{
		rw:            cfg.RW,
		speedInterval: interval,
		onDataHook:    cfg.OnData,
		stopCh:        make(chan struct{}),
	}
	logger.Debug("TrafficMonitor created",
		zap.String("component", "instrument.TrafficMonitor"),
		zap.Duration("speed_interval", m.speedInterval))
	return m
}

// Read implements io.Reader. Counts bytes and optionally calls OnData hook.
func (m *TrafficMonitor) Read(p []byte) (n int, err error) {
	n, err = m.rw.Read(p)
	if n > 0 {
		m.readBytes.Add(int64(n))
		if m.onDataHook != nil {
			m.onDataHook(p[:n])
		}
		logger.Debug("TrafficMonitor Read",
			zap.String("component", "instrument.TrafficMonitor"),
			zap.Int("bytes", n),
			zap.Int64("total_read", m.readBytes.Load()))
	}
	return n, err
}

// Write implements io.Writer. Counts bytes and optionally calls OnData hook.
func (m *TrafficMonitor) Write(p []byte) (n int, err error) {
	n, err = m.rw.Write(p)
	if n > 0 {
		m.writeBytes.Add(int64(n))
		if m.onDataHook != nil {
			m.onDataHook(p[:n])
		}
		logger.Debug("TrafficMonitor Write",
			zap.String("component", "instrument.TrafficMonitor"),
			zap.Int("bytes", n),
			zap.Int64("total_write", m.writeBytes.Load()))
	}
	return n, err
}

// ReadBytes returns total bytes read (atomic).
func (m *TrafficMonitor) ReadBytes() int64 {
	return m.readBytes.Load()
}

// WriteBytes returns total bytes written (atomic).
func (m *TrafficMonitor) WriteBytes() int64 {
	return m.writeBytes.Load()
}

// TotalBytes returns ReadBytes + WriteBytes.
func (m *TrafficMonitor) TotalBytes() int64 {
	return m.readBytes.Load() + m.writeBytes.Load()
}

// Stats returns current byte counts.
func (m *TrafficMonitor) Stats() (read, write int64) {
	return m.readBytes.Load(), m.writeBytes.Load()
}

// SetOnData sets the data hook (thread-safe). Pass nil to clear.
func (m *TrafficMonitor) SetOnData(hook OnDataHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onDataHook = hook
	logger.Debug("TrafficMonitor OnData hook updated",
		zap.String("component", "instrument.TrafficMonitor"),
		zap.Bool("hook_set", hook != nil))
}

// SpeedCalculator computes bytes/sec over a sliding window.
type SpeedCalculator struct {
	samples   []speedSample
	window    time.Duration
	mu        sync.RWMutex
	maxSamples int
}

type speedSample struct {
	bytes int64
	at    time.Time
}

// NewSpeedCalculator creates a calculator with the given window (e.g. 5s).
func NewSpeedCalculator(window time.Duration, maxSamples int) *SpeedCalculator {
	if maxSamples <= 0 {
		maxSamples = 30
	}
	return &SpeedCalculator{
		samples:    make([]speedSample, 0, maxSamples),
		window:     window,
		maxSamples: maxSamples,
	}
}

// Record adds bytes at the current time.
func (sc *SpeedCalculator) Record(bytes int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-sc.window)
	newSamples := make([]speedSample, 0, len(sc.samples)+1)
	for _, s := range sc.samples {
		if s.at.After(cutoff) {
			newSamples = append(newSamples, s)
		}
	}
	newSamples = append(newSamples, speedSample{bytes: bytes, at: now})
	if len(newSamples) > sc.maxSamples {
		newSamples = newSamples[len(newSamples)-sc.maxSamples:]
	}
	sc.samples = newSamples
}

// BytesPerSec returns the average bytes/sec over the window.
func (sc *SpeedCalculator) BytesPerSec() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if len(sc.samples) < 2 {
		return 0
	}
	cutoff := time.Now().Add(-sc.window)
	var total int64
	var first, last time.Time
	for i, s := range sc.samples {
		if s.at.Before(cutoff) {
			continue
		}
		total += s.bytes
		if first.IsZero() || s.at.Before(first) {
			first = s.at
		}
		if last.IsZero() || s.at.After(last) {
			last = s.at
		}
		_ = i
	}
	elapsed := last.Sub(first).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(total) / elapsed
}
