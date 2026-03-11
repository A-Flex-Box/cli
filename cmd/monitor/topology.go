package monitor

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type NetworkNode struct {
	IP          string
	HostName    string
	IsLocal     bool
	IsGateway   bool
	Protocol    string
	Connections []Connection
	BytesIn     int64
	BytesOut    int64
	PacketsIn   int64
	PacketsOut  int64
	LastSeen    time.Time
}

type Connection struct {
	RemoteIP   string
	RemotePort int
	Protocol   string
	State      string
	Bytes      int64
}

type NetworkTopology struct {
	mu           sync.RWMutex
	nodes        map[string]*NetworkNode
	localIPs     []string
	gatewayIP    string
	interfaceIPs map[string]string
}

func NewNetworkTopology() *NetworkTopology {
	t := &NetworkTopology{
		nodes:        make(map[string]*NetworkNode),
		interfaceIPs: make(map[string]string),
	}
	t.discoverLocalIPs()
	return t
}

func (t *NetworkTopology) discoverLocalIPs() {
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					t.localIPs = append(t.localIPs, ipnet.IP.String())
					t.interfaceIPs[ipnet.IP.String()] = iface.Name
				}
			}
		}
	}
}

func (t *NetworkTopology) SetGateway(gatewayIP string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gatewayIP = gatewayIP
	if node, exists := t.nodes[gatewayIP]; exists {
		node.IsGateway = true
	} else {
		t.nodes[gatewayIP] = &NetworkNode{
			IP:        gatewayIP,
			IsGateway: true,
			LastSeen:  time.Now(),
		}
	}
}

func (t *NetworkTopology) UpdateFromPackets(packets []PacketInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, p := range packets {
		t.updateFromPacket(p)
	}
}

func (t *NetworkTopology) updateFromPacket(p PacketInfo) {
	srcIP := p.SrcIP.String()
	dstIP := p.DstIP.String()

	if srcIP == "" || dstIP == "" {
		return
	}

	srcLocal := t.isLocalIP(srcIP)
	dstLocal := t.isLocalIP(dstIP)

	// Update source node
	if srcNode, exists := t.nodes[srcIP]; exists {
		srcNode.BytesOut += int64(p.Length)
		srcNode.PacketsOut++
		srcNode.LastSeen = p.Timestamp
		if p.Protocol != "" {
			srcNode.Protocol = p.Protocol
		}
	} else {
		t.nodes[srcIP] = &NetworkNode{
			IP:         srcIP,
			IsLocal:    srcLocal,
			Protocol:   p.Protocol,
			BytesOut:   int64(p.Length),
			PacketsOut: 1,
			LastSeen:   p.Timestamp,
		}
	}

	// Update destination node
	if dstNode, exists := t.nodes[dstIP]; exists {
		dstNode.BytesIn += int64(p.Length)
		dstNode.PacketsIn++
		dstNode.LastSeen = p.Timestamp
		if p.Protocol != "" && dstNode.Protocol == "" {
			dstNode.Protocol = p.Protocol
		}
	} else {
		t.nodes[dstIP] = &NetworkNode{
			IP:        dstIP,
			IsLocal:   dstLocal,
			Protocol:  p.Protocol,
			BytesIn:   int64(p.Length),
			PacketsIn: 1,
			LastSeen:  p.Timestamp,
		}
	}

	// Add connection from source to destination
	srcNode := t.nodes[srcIP]
	conn := Connection{
		RemoteIP:   dstIP,
		RemotePort: int(p.DstPort),
		Protocol:   p.Protocol,
		Bytes:      int64(p.Length),
	}
	srcNode.Connections = t.addOrUpdateConnection(srcNode.Connections, conn)
}

func (t *NetworkTopology) UpdateFromConnections(conns []ConnTrackEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, c := range conns {
		t.updateFromConnection(c)
	}
}

func (t *NetworkTopology) updateFromConnection(c ConnTrackEntry) {
	srcLocal := t.isLocalIP(c.SrcIP)
	dstLocal := t.isLocalIP(c.DstIP)

	// Update source node
	if srcNode, exists := t.nodes[c.SrcIP]; exists {
		srcNode.BytesOut += c.BytesSrc
		srcNode.PacketsOut += c.PacketsSrc
		srcNode.LastSeen = time.Now()
	} else {
		t.nodes[c.SrcIP] = &NetworkNode{
			IP:         c.SrcIP,
			IsLocal:    srcLocal,
			Protocol:   c.Proto,
			BytesOut:   c.BytesSrc,
			PacketsOut: c.PacketsSrc,
			LastSeen:   time.Now(),
		}
	}

	// Update destination node
	if dstNode, exists := t.nodes[c.DstIP]; exists {
		dstNode.BytesIn += c.BytesDst
		dstNode.PacketsIn += c.PacketsDst
		dstNode.LastSeen = time.Now()
	} else {
		t.nodes[c.DstIP] = &NetworkNode{
			IP:        c.DstIP,
			IsLocal:   dstLocal,
			Protocol:  c.Proto,
			BytesIn:   c.BytesDst,
			PacketsIn: c.PacketsDst,
			LastSeen:  time.Now(),
		}
	}

	// Update connection with state
	srcNode := t.nodes[c.SrcIP]
	conn := Connection{
		RemoteIP:   c.DstIP,
		RemotePort: c.DstPort,
		Protocol:   c.Proto,
		State:      c.State,
		Bytes:      c.BytesSrc + c.BytesDst,
	}
	srcNode.Connections = t.addOrUpdateConnection(srcNode.Connections, conn)
}

func (t *NetworkTopology) addOrUpdateConnection(conns []Connection, newConn Connection) []Connection {
	for i, c := range conns {
		if c.RemoteIP == newConn.RemoteIP && c.RemotePort == newConn.RemotePort && c.Protocol == newConn.Protocol {
			conns[i].Bytes += newConn.Bytes
			if newConn.State != "" {
				conns[i].State = newConn.State
			}
			return conns
		}
	}
	return append(conns, newConn)
}

func (t *NetworkTopology) isLocalIP(ip string) bool {
	for _, localIP := range t.localIPs {
		if ip == localIP {
			return true
		}
	}
	return false
}

func (t *NetworkTopology) GetNodes() []*NetworkNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nodes := make([]*NetworkNode, 0, len(t.nodes))
	for _, node := range t.nodes {
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].BytesIn+nodes[i].BytesOut > nodes[j].BytesIn+nodes[j].BytesOut
	})

	return nodes
}

func (t *NetworkTopology) GetLocalNodes() []*NetworkNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nodes := make([]*NetworkNode, 0)
	for _, node := range t.nodes {
		if node.IsLocal || node.IsGateway {
			nodes = append(nodes, node)
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsGateway != nodes[j].IsGateway {
			return nodes[i].IsGateway
		}
		return nodes[i].BytesIn+nodes[i].BytesOut > nodes[j].BytesIn+nodes[j].BytesOut
	})

	return nodes
}

func (t *NetworkTopology) GetTopologyGraph() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var sb strings.Builder

	localStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)

	remoteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00BFFF"))

	gatewayStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true)

	connectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1)

	sb.WriteString(headerStyle.Render("Network Topology"))
	sb.WriteString("\n\n")

	sb.WriteString(localStyle.Render("● Local Host"))
	sb.WriteString("  ")
	sb.WriteString(gatewayStyle.Render("★ Gateway"))
	sb.WriteString("  ")
	sb.WriteString(remoteStyle.Render("○ Remote"))
	sb.WriteString("\n\n")

	if len(t.localIPs) > 0 {
		sb.WriteString(localStyle.Render("┌─ Local Interfaces"))
		sb.WriteString("\n")
		for i, ip := range t.localIPs {
			iface := t.interfaceIPs[ip]
			prefix := "├── "
			if i == len(t.localIPs)-1 {
				prefix = "└── "
			}
			sb.WriteString(fmt.Sprintf("│  %s%s (%s)\n", prefix, ip, iface))
		}
		sb.WriteString("│\n")
	}

	if t.gatewayIP != "" {
		sb.WriteString(connectionStyle.Render("│"))
		sb.WriteString("\n")
		sb.WriteString(gatewayStyle.Render(fmt.Sprintf("★ Gateway: %s", t.gatewayIP)))
		sb.WriteString("\n")
		sb.WriteString(connectionStyle.Render("│"))
		sb.WriteString("\n")
	}

	topNodes := t.GetNodes()
	if len(topNodes) > 20 {
		topNodes = topNodes[:20]
	}

	sb.WriteString(connectionStyle.Render("├─ Active Connections (Top 20)"))
	sb.WriteString("\n")

	for _, node := range topNodes {
		if node.IsLocal && !node.IsGateway {
			continue
		}

		var style lipgloss.Style
		var marker string

		switch {
		case node.IsGateway:
			style = gatewayStyle
			marker = "★"
		case node.IsLocal:
			style = localStyle
			marker = "●"
		default:
			style = remoteStyle
			marker = "○"
		}

		sb.WriteString(fmt.Sprintf("│  %s %s - In: %s Out: %s\n",
			style.Render(marker),
			style.Render(node.IP),
			formatBytes(node.BytesIn),
			formatBytes(node.BytesOut)))
		sb.WriteString(fmt.Sprintf("│      Proto: %s Pkts: %d/%d\n",
			node.Protocol,
			node.PacketsIn,
			node.PacketsOut))
	}

	return sb.String()
}

func (t *NetworkTopology) RenderCompact(width int) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Padding(0, 1)

	sb.WriteString(headerStyle.Render("Network Topology"))
	sb.WriteString("\n\n")

	localCount := 0
	remoteCount := 0
	for _, node := range t.nodes {
		if node.IsLocal {
			localCount++
		} else {
			remoteCount++
		}
	}

	sb.WriteString(boxStyle.Render(fmt.Sprintf(
		"Local: %d | Remote: %d | Total: %d",
		localCount, remoteCount, len(t.nodes))))
	sb.WriteString("\n\n")

	topNodes := t.GetNodes()
	if len(topNodes) > 10 {
		topNodes = topNodes[:10]
	}

	for _, node := range topNodes {
		if node.IsLocal && !node.IsGateway {
			continue
		}

		marker := "○"
		if node.IsGateway {
			marker = "★"
		} else if node.IsLocal {
			marker = "●"
		}

		line := fmt.Sprintf("%s %s - %s in / %s out",
			marker, node.IP,
			formatBytes(node.BytesIn),
			formatBytes(node.BytesOut))

		if len(line) > width-4 {
			line = line[:width-7] + "..."
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func (t *NetworkTopology) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes = make(map[string]*NetworkNode)
}
