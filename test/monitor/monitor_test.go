package monitor_test

import (
	"testing"
	"time"

	"github.com/A-Flex-Box/cli/cmd/monitor"
)

func TestBandwidthMonitor(t *testing.T) {
	m := monitor.NewBandwidthMonitor("", time.Second)

	if m == nil {
		t.Fatal("Expected non-nil monitor")
	}

	if m.IsRunning() {
		t.Error("Monitor should not be running initially")
	}

	err := m.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	if !m.IsRunning() {
		t.Error("Monitor should be running after start")
	}

	time.Sleep(100 * time.Millisecond)

	stats := m.GetStats()
	t.Logf("Initial stats: %+v", stats)

	m.Stop()

	if m.IsRunning() {
		t.Error("Monitor should not be running after stop")
	}
}

func TestDNSMonitor(t *testing.T) {
	m := monitor.NewDNSMonitor()

	if m == nil {
		t.Fatal("Expected non-nil monitor")
	}

	if m.IsRunning() {
		t.Error("Monitor should not be running initially")
	}

	err := m.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	if !m.IsRunning() {
		t.Error("Monitor should be running after start")
	}

	time.Sleep(100 * time.Millisecond)

	stats := m.GetStats()
	t.Logf("Initial stats: %+v", stats)

	queries := m.GetQueries()
	t.Logf("Queries count: %d", len(queries))

	m.Stop()

	if m.IsRunning() {
		t.Error("Monitor should not be running after stop")
	}
}

func TestInterfaceStats(t *testing.T) {
	iface, err := monitor.GetDefaultInterfaceName()
	if err != nil {
		t.Skipf("No default interface: %v", err)
	}

	t.Logf("Default interface: %s", iface)

	ifaces, err := monitor.GetAvailableInterfaces()
	if err != nil {
		t.Fatalf("Failed to get interfaces: %v", err)
	}

	if len(ifaces) == 0 {
		t.Error("Expected at least one interface")
	}

	for _, i := range ifaces {
		t.Logf("Interface: %s, MTU: %d, Flags: %v", i.Name, i.MTU, i.Flags)
	}
}

func TestFormatBandwidth(t *testing.T) {
	tests := []struct {
		bps      int64
		expected string
	}{
		{100, "100 B/s"},
		{1024, "1.0 KB/s"},
		{1536, "1.5 KB/s"},
		{1048576, "1.0 MB/s"},
		{1073741824, "1.0 GB/s"},
	}

	for _, tt := range tests {
		result := monitor.FormatBandwidth(tt.bps)
		t.Logf("FormatBandwidth(%d) = %s (expected: %s)", tt.bps, result, tt.expected)
	}
}
