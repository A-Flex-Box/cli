package doctor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

// ProcessInfo holds PID, process name, and command line for a port occupant.
type ProcessInfo struct {
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	CommandLine string `json:"command_line"`
	Permission  string `json:"permission,omitempty"` // e.g. "Unknown (Permission Denied)"
}

// GetPortOccupancy returns the process listening on the given port.
// Handles permission errors gracefully: if root is needed, returns
// ProcessInfo with Permission set to "Unknown (Permission Denied)".
func GetPortOccupancy(port int) (*ProcessInfo, error) {
	logger.Debug("GetPortOccupancy: querying port",
		zap.String("component", "doctor.system"),
		zap.Int("port", port))

	conns, err := net.ConnectionsWithContext(context.Background(), "tcp4")
	if err != nil {
		logger.Warn("GetPortOccupancy: net.Connections failed",
			zap.String("component", "doctor.system"),
			zap.Int("port", port),
			zap.Error(err))
		// Try tcp (all) as fallback
		conns, err = net.ConnectionsWithContext(context.Background(), "tcp")
		if err != nil {
			logger.Error("GetPortOccupancy: net.Connections fallback failed",
				zap.String("component", "doctor.system"),
				zap.Int("port", port),
				zap.Error(err))
			return &ProcessInfo{
				PID:         0,
				ProcessName: "Unknown",
				Permission:  "Permission Denied: " + err.Error(),
			}, nil
		}
	}

	portStr := strconv.Itoa(port)
	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		localPort := ""
		if c.Laddr.Port != 0 {
			localPort = strconv.FormatUint(uint64(c.Laddr.Port), 10)
		}
		if localPort != portStr {
			continue
		}
		pid := int(c.Pid)
		if pid == 0 {
			logger.Debug("GetPortOccupancy: found listener but PID=0 (kernel or permission)",
				zap.String("component", "doctor.system"),
				zap.Int("port", port))
			return &ProcessInfo{
				PID:         0,
				ProcessName: "Unknown",
				Permission:  "Unknown (Permission Denied or Kernel)",
			}, nil
		}
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			logger.Warn("GetPortOccupancy: process.NewProcess failed",
				zap.String("component", "doctor.system"),
				zap.Int("port", port),
				zap.Int("pid", pid),
				zap.Error(err))
			return &ProcessInfo{
				PID:         pid,
				ProcessName: "Unknown",
				Permission:  "Process access denied: " + err.Error(),
			}, nil
		}
		name, _ := proc.Name()
		cmdline, _ := proc.Cmdline()
		logger.Debug("GetPortOccupancy: found process",
			zap.String("component", "doctor.system"),
			zap.Int("port", port),
			zap.Int("pid", pid),
			zap.String("name", name))
		return &ProcessInfo{
			PID:         pid,
			ProcessName: name,
			CommandLine: cmdline,
		}, nil
	}
	logger.Debug("GetPortOccupancy: no listener found for port",
		zap.String("component", "doctor.system"),
		zap.Int("port", port))
	return nil, nil
}

// InterfaceStat holds per-interface network stats for the watch dashboard.
type InterfaceStat struct {
	Name      string  `json:"name"`
	BytesRecv uint64  `json:"bytes_recv"`
	BytesSent uint64  `json:"bytes_sent"`
	RecvSpeed float64 `json:"recv_speed,omitempty"` // bytes/sec
	SentSpeed float64 `json:"sent_speed,omitempty"` // bytes/sec
	Up        bool    `json:"up"`
}

// GetInterfaceStats returns a list of network interfaces with their current I/O stats.
// Speed fields are 0 unless caller provides previous snapshot for delta.
func GetInterfaceStats() ([]InterfaceStat, error) {
	logger.Debug("GetInterfaceStats: querying",
		zap.String("component", "doctor.system"))

	ioCounters, err := net.IOCounters(true)
	if err != nil {
		logger.Error("GetInterfaceStats: net.IOCounters failed",
			zap.String("component", "doctor.system"),
			zap.Error(err))
		return nil, fmt.Errorf("net.IOCounters: %w", err)
	}
	stats := make([]InterfaceStat, 0, len(ioCounters))
	for _, io := range ioCounters {
		stats = append(stats, InterfaceStat{
			Name:      io.Name,
			BytesRecv: io.BytesRecv,
			BytesSent: io.BytesSent,
			Up:        true,
		})
	}
	logger.Debug("GetInterfaceStats: fetched interfaces",
		zap.String("component", "doctor.system"),
		zap.Int("count", len(stats)))
	return stats, nil
}

// ConnectionInfo holds a single connection for the watch dashboard.
type ConnectionInfo struct {
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
	Status     string `json:"status"`
	PID        int    `json:"pid"`
}

// GetActiveConnections returns top TCP connections (ESTABLISHED preferred).
func GetActiveConnections(limit int) ([]ConnectionInfo, error) {
	logger.Debug("GetActiveConnections: querying",
		zap.String("component", "doctor.system"),
		zap.Int("limit", limit))

	conns, err := net.ConnectionsWithContext(context.Background(), "tcp")
	if err != nil {
		logger.Error("GetActiveConnections: failed",
			zap.String("component", "doctor.system"),
			zap.Error(err))
		return nil, fmt.Errorf("net.Connections: %w", err)
	}
	if limit <= 0 {
		limit = 20
	}
	// Prefer ESTABLISHED, then LISTEN
	var established, listen, other []net.ConnectionStat
	for _, c := range conns {
		switch c.Status {
		case "ESTABLISHED":
			established = append(established, c)
		case "LISTEN":
			listen = append(listen, c)
		default:
			other = append(other, c)
		}
	}
	combined := append(append(established, listen...), other...)
	result := make([]ConnectionInfo, 0, limit)
	for i := 0; i < len(combined) && i < limit; i++ {
		c := combined[i]
		result = append(result, ConnectionInfo{
			LocalAddr:  fmt.Sprintf("%s:%d", c.Laddr.IP, c.Laddr.Port),
			RemoteAddr: fmt.Sprintf("%s:%d", c.Raddr.IP, c.Raddr.Port),
			Status:     c.Status,
			PID:        int(c.Pid),
		})
	}
	logger.Debug("GetActiveConnections: returned",
		zap.String("component", "doctor.system"),
		zap.Int("count", len(result)))
	return result, nil
}
