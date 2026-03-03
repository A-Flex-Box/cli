package wormhole

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/hashicorp/yamux"
)

// UIEventType identifies the kind of tunnel UI event.
type UIEventType int

const (
	EventConnOpen UIEventType = iota
	EventConnClose
	EventTraffic
)

// UIEvent is sent to the tunnel TUI for display.
type UIEvent struct {
	Type   UIEventType
	Msg    string       // Display string
	Info   *TrafficInfo // For EventTraffic
	Remote string       // Optional: remote addr
}

// TunnelOptions holds optional settings for tunnel UI events.
type TunnelOptions struct {
	Events chan<- UIEvent // If non-nil, tunnel sends UI events here
}

// ExposeTunnel dials relay, performs PAKE (sender), sends ModeTunnel, then runs StartExpose.
// Blocks until the tunnel is closed. opts may be nil.
func ExposeTunnel(relayAddr, code, targetPort string, opts *TunnelOptions) error {
	logger.Info("tunnel.expose DialRelay", logger.Context("params", map[string]any{"relay": relayAddr, "code": code})...)
	conn, err := DialRelay(relayAddr, code, true)
	if err != nil {
		logger.Warn("tunnel.expose DialRelay failed", logger.Context("params", map[string]any{"error": err.Error()})...)
		return err
	}
	defer conn.Close()

	logger.Info("tunnel.expose PAKE upgrade (sender)", logger.Context("params", map[string]any{"remote": conn.RemoteAddr().String()})...)
	secure, err := UpgradeConn(conn, code, true)
	if err != nil {
		logger.Warn("tunnel.expose UpgradeConn failed", logger.Context("params", map[string]any{"error": err.Error()})...)
		return err
	}
	defer secure.Close()

	if _, err := secure.Write([]byte{ModeTunnel}); err != nil {
		logger.Warn("tunnel.expose write ModeTunnel failed", logger.Context("params", map[string]any{"error": err.Error()})...)
		return err
	}
	logger.Info("tunnel.expose mode sent, starting yamux server", logger.Context("params", map[string]any{"target_port": targetPort})...)
	return StartExpose(secure, targetPort, opts)
}

// ConnectTunnel dials relay, performs PAKE (receiver), reads mode byte, then runs StartConnect.
// If mode is ModeFile, returns an error. Blocks until the tunnel is closed. opts may be nil.
func ConnectTunnel(relayAddr, code, bindAddr string, opts *TunnelOptions) error {
	logger.Info("tunnel.connect DialRelay", logger.Context("params", map[string]any{"relay": relayAddr, "code": code})...)
	conn, err := DialRelay(relayAddr, code, false)
	if err != nil {
		logger.Warn("tunnel.connect DialRelay failed", logger.Context("params", map[string]any{"error": err.Error()})...)
		return err
	}
	defer conn.Close()

	logger.Info("tunnel.connect PAKE upgrade (receiver)", logger.Context("params", map[string]any{"remote": conn.RemoteAddr().String()})...)
	secure, err := UpgradeConn(conn, code, false)
	if err != nil {
		logger.Warn("tunnel.connect UpgradeConn failed", logger.Context("params", map[string]any{"error": err.Error()})...)
		return err
	}
	defer secure.Close()

	mode := make([]byte, 1)
	if _, err := io.ReadFull(secure, mode); err != nil {
		logger.Warn("tunnel.connect read mode failed", logger.Context("params", map[string]any{"error": err.Error()})...)
		return err
	}
	if mode[0] == ModeFile {
		return fmt.Errorf("peer is in file transfer mode, not tunnel mode")
	}
	if mode[0] != ModeTunnel {
		return fmt.Errorf("unknown mode byte: %d", mode[0])
	}
	logger.Info("tunnel.connect mode received, starting yamux client", logger.Context("params", map[string]any{"bind_addr": bindAddr})...)
	return StartConnect(secure, bindAddr, opts)
}

// yamuxConfig enables keepalive to prevent Relay/NAT from killing idle connections.
var yamuxConfig = func() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = 30 * time.Second
	return cfg
}()

// StartExpose creates a yamux server on the secure connection and forwards
// incoming streams to the local targetPort. Blocks until secureConn is closed.
// opts may be nil.
func StartExpose(secureConn net.Conn, targetPort string, opts *TunnelOptions) error {
	session, err := yamux.Server(secureConn, yamuxConfig)
	if err != nil {
		return err
	}
	defer session.Close()

	ev := evChan(opts)
	logger.Info("tunnel.expose session started", logger.Context("params", map[string]any{
		"target_port": targetPort,
	})...)

	for {
		stream, err := session.Accept()
		if err != nil {
			if session.IsClosed() {
				logger.Info("tunnel.expose session closed (secure conn closed by peer or network)")
				return nil
			}
			logger.Warn("tunnel.expose accept error", logger.Context("params", map[string]any{"error": err.Error(), "session_closed": session.IsClosed()})...)
			return err
		}

		destAddr := "localhost:" + targetPort
		destConn, err := net.Dial("tcp", destAddr)
		if err != nil {
			sendEvent(ev, UIEvent{Type: EventTraffic, Msg: "[FAIL] dial " + destAddr + ": " + err.Error()})
			logger.Info("tunnel.expose dial target failed", logger.Context("params", map[string]any{
				"target": destAddr, "error": err.Error(),
			})...)
			stream.Close()
			continue
		}

		sendEvent(ev, UIEvent{Type: EventConnOpen, Msg: "Client Connected", Remote: destAddr})
		logger.Info("tunnel.expose stream forwarded", logger.Context("params", map[string]any{
			"target": destAddr,
		})...)
		go join(stream, destConn, ev)
	}
}

func evChan(opts *TunnelOptions) chan<- UIEvent {
	if opts == nil {
		return nil
	}
	return opts.Events
}

func sendEvent(ch chan<- UIEvent, e UIEvent) {
	if ch == nil {
		return
	}
	select {
	case ch <- e:
	default:
		// non-blocking; drop if full
	}
}

// StartConnect creates a yamux client on the secure connection, listens on
// bindAddr, and for each local connection opens a new stream and joins.
// Blocks until secureConn is closed. opts may be nil.
func StartConnect(secureConn net.Conn, bindAddr string, opts *TunnelOptions) error {
	session, err := yamux.Client(secureConn, yamuxConfig)
	if err != nil {
		return err
	}
	defer session.Close()

	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	ev := evChan(opts)
	logger.Info("tunnel.connect listening", logger.Context("params", map[string]any{
		"bind_addr": bindAddr,
	})...)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			logger.Info("tunnel.connect accept error", logger.Context("params", map[string]any{"error": err.Error()})...)
			return err
		}

		remote := localConn.RemoteAddr().String()
		sendEvent(ev, UIEvent{Type: EventConnOpen, Msg: "Client Connected", Remote: remote})
		logger.Info("tunnel.connect local connection accepted", logger.Context("params", map[string]any{
			"remote": remote,
		})...)

		stream, err := session.Open()
		if err != nil {
			sendEvent(ev, UIEvent{Type: EventTraffic, Msg: "[FAIL] open stream: " + err.Error()})
			logger.Info("tunnel.connect open stream failed", logger.Context("params", map[string]any{
				"error": err.Error(),
			})...)
			localConn.Close()
			continue
		}

		logger.Info("tunnel.connect forwarding", logger.Context("params", map[string]any{
			"local": remote,
		})...)
		go join(localConn, stream, ev)
	}
}

// join performs bidirectional copy between two connections with optional traffic sniffing.
// Closes both when either side finishes. events may be nil.
func join(c1, c2 net.Conn, events chan<- UIEvent) {
	defer c1.Close()
	defer c2.Close()
	defer func() { sendEvent(events, UIEvent{Type: EventConnClose, Msg: "Client Disconnected"}) }()

	var src1 io.Reader = c1
	if events != nil {
		src1 = NewSniffingReader(c1, func(info TrafficInfo) {
			sendEvent(events, UIEvent{Type: EventTraffic, Msg: info.Raw, Info: &info})
		})
	}

	done := make(chan struct{}, 1)
	go func() {
		io.Copy(c2, src1)
		if tcp, ok := c2.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		io.Copy(c1, c2)
		if tcp, ok := c1.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
		done <- struct{}{}
	}()
	<-done
}
