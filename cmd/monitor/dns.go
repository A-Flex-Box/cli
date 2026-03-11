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
)

type DNSQuery struct {
	Timestamp   time.Time
	SrcIP       string
	DstIP       string
	SrcPort     int
	DstPort     int
	QueryType   string
	Domain      string
	ResponseIPs []string
	Latency     time.Duration
	IsResponse  bool
}

type DNSMonitor struct {
	mu          sync.RWMutex
	queries     []DNSQuery
	domains     map[string]int
	maxQueries  int
	events      chan DNSQuery
	stopChan    chan struct{}
	running     bool
	refreshRate time.Duration

	port53conns map[string]time.Time
}

func NewDNSMonitor() *DNSMonitor {
	return &DNSMonitor{
		queries:     make([]DNSQuery, 0, 500),
		domains:     make(map[string]int),
		maxQueries:  500,
		events:      make(chan DNSQuery, 50),
		stopChan:    make(chan struct{}),
		refreshRate: time.Second,
		port53conns: make(map[string]time.Time),
	}
}

func (m *DNSMonitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.running = true
	go m.monitorLoop()

	return nil
}

func (m *DNSMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopChan)
	m.running = false
}

func (m *DNSMonitor) Events() <-chan DNSQuery {
	return m.events
}

func (m *DNSMonitor) GetQueries() []DNSQuery {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]DNSQuery, len(m.queries))
	copy(result, m.queries)
	return result
}

func (m *DNSMonitor) GetTopDomains(n int) []DomainStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var stats []DomainStats
	for domain, count := range m.domains {
		stats = append(stats, DomainStats{
			Domain: domain,
			Count:  count,
		})
	}

	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Count > stats[i].Count {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	if len(stats) > n {
		stats = stats[:n]
	}

	return stats
}

type DomainStats struct {
	Domain string
	Count  int
}

func (m *DNSMonitor) monitorLoop() {
	ticker := time.NewTicker(m.refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkDNS()
		}
	}
}

func (m *DNSMonitor) checkDNS() {
	m.checkDNSFromProc()
}

func (m *DNSMonitor) checkDNSFromProc() {
	entries := m.readDNSConnections()
	m.processDNSConnections(entries)
}

func (m *DNSMonitor) readDNSConnections() []DNSQuery {
	var queries []DNSQuery

	queries = append(queries, m.readDNSFromProcFile("/proc/net/udp")...)
	queries = append(queries, m.readDNSFromProcFile("/proc/net/udp6")...)

	return queries
}

func (m *DNSMonitor) readDNSFromProcFile(filepath string) []DNSQuery {
	file, err := os.Open(filepath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var queries []DNSQuery
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

		srcIP, srcPort := parseDNSAddr(localAddr)
		dstIP, dstPort := parseDNSAddr(remoteAddr)

		if srcIP == "" || dstIP == "" {
			continue
		}

		if srcPort == 53 || dstPort == 53 {
			queries = append(queries, DNSQuery{
				Timestamp:  time.Now(),
				SrcIP:      srcIP,
				DstIP:      dstIP,
				SrcPort:    srcPort,
				DstPort:    dstPort,
				IsResponse: srcPort == 53,
			})
		}
	}

	return queries
}

func parseDNSAddr(addr string) (string, int) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0
	}

	ipHex := parts[0]
	portHex := parts[1]

	port, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return "", 0
	}

	var ip net.IP
	if len(ipHex) == 8 {
		ip = make(net.IP, 4)
		for i := 0; i < 4; i++ {
			start := 6 - 2*i
			end := 8 - 2*i
			if start < 0 || end > len(ipHex) {
				return "", 0
			}
			b, err := hex.DecodeString(ipHex[start:end])
			if err != nil || len(b) == 0 {
				return "", 0
			}
			ip[i] = b[0]
		}
	} else if len(ipHex) == 32 {
		ip = make(net.IP, 16)
		for i := 0; i < 16; i++ {
			start := 28 - 2*i
			end := 30 - 2*i
			if start < 0 || end > len(ipHex) {
				return "", 0
			}
			b, err := hex.DecodeString(ipHex[start:end])
			if err != nil || len(b) == 0 {
				return "", 0
			}
			ip[i] = b[0]
		}
	} else {
		return "", 0
	}

	return ip.String(), int(port)
}

func (m *DNSMonitor) processDNSConnections(queries []DNSQuery) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, q := range queries {
		connKey := fmt.Sprintf("%s:%d->%s:%d", q.SrcIP, q.SrcPort, q.DstIP, q.DstPort)

		if !q.IsResponse {
			if _, exists := m.port53conns[connKey]; !exists {
				m.port53conns[connKey] = q.Timestamp
				m.queries = append(m.queries, q)

				if len(m.queries) > m.maxQueries {
					m.queries = m.queries[len(m.queries)-m.maxQueries:]
				}

				select {
				case m.events <- q:
				default:
				}
			}
		} else {
			queryKey := fmt.Sprintf("%s:%d->%s:%d", q.DstIP, q.DstPort, q.SrcIP, 53)
			if queryTime, exists := m.port53conns[queryKey]; exists {
				latency := q.Timestamp.Sub(queryTime)

				for i := len(m.queries) - 1; i >= 0; i-- {
					if m.queries[i].SrcIP == q.DstIP && m.queries[i].DstPort == 53 {
						m.queries[i].Latency = latency
						m.queries[i].ResponseIPs = []string{q.SrcIP}
						break
					}
				}

				delete(m.port53conns, queryKey)
			}
		}
	}
}

func (m *DNSMonitor) AddQuery(q DNSQuery) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queries = append(m.queries, q)
	if len(m.queries) > m.maxQueries {
		m.queries = m.queries[len(m.queries)-m.maxQueries:]
	}

	if q.Domain != "" {
		m.domains[q.Domain]++
	}

	select {
	case m.events <- q:
	default:
	}
}

func (m *DNSMonitor) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queries = make([]DNSQuery, 0, m.maxQueries)
	m.domains = make(map[string]int)
	m.port53conns = make(map[string]time.Time)
}

func (m *DNSMonitor) GetStats() DNSMonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return DNSMonitorStats{
		TotalQueries:   len(m.queries),
		UniqueDomains:  len(m.domains),
		QueriesPerSec:  m.calculateQPS(),
		AvgLatency:     m.calculateAvgLatency(),
		PendingQueries: len(m.port53conns),
	}
}

type DNSMonitorStats struct {
	TotalQueries   int
	UniqueDomains  int
	QueriesPerSec  float64
	AvgLatency     time.Duration
	PendingQueries int
}

func (m *DNSMonitor) calculateQPS() float64 {
	if len(m.queries) < 2 {
		return 0
	}

	first := m.queries[0].Timestamp
	last := m.queries[len(m.queries)-1].Timestamp
	duration := last.Sub(first).Seconds()

	if duration <= 0 {
		return 0
	}

	return float64(len(m.queries)) / duration
}

func (m *DNSMonitor) calculateAvgLatency() time.Duration {
	var total time.Duration
	var count int

	for _, q := range m.queries {
		if q.Latency > 0 {
			total += q.Latency
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

func (m *DNSMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func ParseSimpleDNSQuery(data []byte) (domain string, qtype string, err error) {
	if len(data) < 12 {
		return "", "", fmt.Errorf("packet too short")
	}

	flags := uint16(data[2])<<8 | uint16(data[3])
	isResponse := (flags & 0x8000) != 0
	if isResponse {
		return "", "", fmt.Errorf("response packet")
	}

	qdcount := int(uint16(data[4])<<8 | uint16(data[5]))
	if qdcount == 0 {
		return "", "", fmt.Errorf("no questions")
	}

	pos := 12
	var labels []string

	for pos < len(data) {
		length := int(data[pos])
		if length == 0 {
			pos++
			break
		}

		if pos+1+length > len(data) {
			break
		}

		labels = append(labels, string(data[pos+1:pos+1+length]))
		pos += 1 + length
	}

	domain = strings.Join(labels, ".")

	if pos+4 <= len(data) {
		qtypeCode := int(uint16(data[pos])<<8 | uint16(data[pos+1]))
		qtype = dnsTypeToString(qtypeCode)
	}

	return domain, qtype, nil
}

func dnsTypeToString(code int) string {
	types := map[int]string{
		1:   "A",
		2:   "NS",
		5:   "CNAME",
		6:   "SOA",
		12:  "PTR",
		15:  "MX",
		16:  "TXT",
		28:  "AAAA",
		33:  "SRV",
		255: "ANY",
	}

	if t, ok := types[code]; ok {
		return t
	}
	return fmt.Sprintf("TYPE%d", code)
}
