package monitor

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

type PacketInfo struct {
	Timestamp time.Time
	SrcIP     net.IP
	DstIP     net.IP
	SrcPort   uint16
	DstPort   uint16
	Protocol  string
	Length    int
	Info      string
}

type Capture struct {
	mu         sync.RWMutex
	iface      string
	filter     string
	packets    []PacketInfo
	maxPackets int
	events     chan PacketInfo
	stopChan   chan struct{}
	running    bool
	stats      CaptureStats
}

type CaptureStats struct {
	TotalPackets int
	TotalBytes   int64
	ByProtocol   map[string]int
}

func NewCapture(iface, bpfFilter string) *Capture {
	return &Capture{
		iface:      iface,
		filter:     bpfFilter,
		packets:    make([]PacketInfo, 0, 1000),
		maxPackets: 1000,
		events:     make(chan PacketInfo, 100),
		stopChan:   make(chan struct{}),
		stats: CaptureStats{
			ByProtocol: make(map[string]int),
		},
	}
}

func (c *Capture) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	c.running = true
	go c.captureLoop()

	logger.Debug("packet capture started (reading from /proc)", zap.String("interface", c.iface))
	return nil
}

func (c *Capture) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	close(c.stopChan)
	c.running = false

	logger.Debug("packet capture stopped")
}

func (c *Capture) Events() <-chan PacketInfo {
	return c.events
}

func (c *Capture) GetPackets() []PacketInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]PacketInfo, len(c.packets))
	copy(result, c.packets)
	return result
}

func (c *Capture) GetStats() CaptureStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *Capture) captureLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.readProcNet()
		}
	}
}

func (c *Capture) readProcNet() {
	c.readProcNetTCP()
	c.readProcNetUDP()
}

func (c *Capture) readProcNetTCP() {
	c.readProcNetFile("/proc/net/tcp", "TCP")
	c.readProcNetFile("/proc/net/tcp6", "TCP6")
}

func (c *Capture) readProcNetUDP() {
	c.readProcNetFile("/proc/net/udp", "UDP")
	c.readProcNetFile("/proc/net/udp6", "UDP6")
}

func (c *Capture) readProcNetFile(filepath, proto string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		localAddr := fields[1]
		remoteAddr := fields[2]

		srcIP, srcPort := parseAddr(localAddr)
		dstIP, dstPort := parseAddr(remoteAddr)

		if srcIP == nil || dstIP == nil {
			continue
		}

		info := PacketInfo{
			Timestamp: time.Now(),
			SrcIP:     srcIP,
			DstIP:     dstIP,
			SrcPort:   srcPort,
			DstPort:   dstPort,
			Protocol:  proto,
			Length:    0,
			Info:      fmt.Sprintf("%s:%d -> %s:%d (%s)", srcIP, srcPort, dstIP, dstPort, proto),
		}

		c.mu.Lock()
		c.packets = append(c.packets, info)
		if len(c.packets) > c.maxPackets {
			c.packets = c.packets[len(c.packets)-c.maxPackets:]
		}
		c.stats.TotalPackets++
		c.stats.ByProtocol[proto]++
		c.mu.Unlock()

		select {
		case c.events <- info:
		default:
		}
	}
}

func parseAddr(addr string) (net.IP, uint16) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return nil, 0
	}

	ipHex := parts[0]
	portHex := parts[1]

	port, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return nil, 0
	}

	var ip net.IP
	if len(ipHex) == 8 {
		ip = make(net.IP, 4)
		for i := 0; i < 4; i++ {
			start := 6 - 2*i
			end := 8 - 2*i
			if start < 0 || end > len(ipHex) {
				return nil, 0
			}
			b, err := hex.DecodeString(ipHex[start:end])
			if err != nil || len(b) == 0 {
				return nil, 0
			}
			ip[i] = b[0]
		}
	} else if len(ipHex) == 32 {
		ip = make(net.IP, 16)
		for i := 0; i < 16; i++ {
			start := 28 - 2*i
			end := 30 - 2*i
			if start < 0 || end > len(ipHex) {
				return nil, 0
			}
			b, err := hex.DecodeString(ipHex[start:end])
			if err != nil || len(b) == 0 {
				return nil, 0
			}
			ip[i] = b[0]
		}
	} else {
		return nil, 0
	}

	return ip, uint16(port)
}

func (c *Capture) getDefaultInterface() (string, error) {
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
