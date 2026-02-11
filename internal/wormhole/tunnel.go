package wormhole

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/hashicorp/yamux"
	"go.uber.org/zap"
)

// ExposeTunnel dials relay, performs PAKE (sender), sends ModeTunnel, then runs StartExpose.
// Blocks until the tunnel is closed.
func ExposeTunnel(relayAddr, code, targetPort string) error {
	conn, err := DialRelay(relayAddr, code, true)
	if err != nil {
		return err
	}
	defer conn.Close()

	secure, err := UpgradeConn(conn, code, true)
	if err != nil {
		return err
	}
	defer secure.Close()

	if _, err := secure.Write([]byte{ModeTunnel}); err != nil {
		return err
	}
	logger.Info("tunnel.expose mode sent, starting yamux server")
	return StartExpose(secure, targetPort)
}

// ConnectTunnel dials relay, performs PAKE (receiver), reads mode byte, then runs StartConnect.
// If mode is ModeFile, returns an error. Blocks until the tunnel is closed.
func ConnectTunnel(relayAddr, code, bindAddr string) error {
	conn, err := DialRelay(relayAddr, code, false)
	if err != nil {
		return err
	}
	defer conn.Close()

	secure, err := UpgradeConn(conn, code, false)
	if err != nil {
		return err
	}
	defer secure.Close()

	mode := make([]byte, 1)
	if _, err := io.ReadFull(secure, mode); err != nil {
		return err
	}
	if mode[0] == ModeFile {
		return fmt.Errorf("peer is in file transfer mode, not tunnel mode")
	}
	if mode[0] != ModeTunnel {
		return fmt.Errorf("unknown mode byte: %d", mode[0])
	}
	logger.Info("tunnel.connect mode received, starting yamux client")
	return StartConnect(secure, bindAddr)
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
func StartExpose(secureConn net.Conn, targetPort string) error {
	session, err := yamux.Server(secureConn, yamuxConfig)
	if err != nil {
		return err
	}
	defer session.Close()

	logger.Info("tunnel.expose session started", logger.Context("params", map[string]any{
		"target_port": targetPort,
	})...)

	for {
		stream, err := session.Accept()
		if err != nil {
			if session.IsClosed() {
				logger.Debug("tunnel.expose session closed")
				return nil
			}
			logger.Warn("tunnel.expose accept error", zap.Error(err))
			return err
		}

		destAddr := "localhost:" + targetPort
		destConn, err := net.Dial("tcp", destAddr)
		if err != nil {
			logger.Warn("tunnel.expose dial target failed", zap.Error(err), zap.String("target", destAddr))
			stream.Close()
			continue
		}

		logger.Debug("tunnel.expose stream accepted, joining", logger.Context("params", map[string]any{
			"target": destAddr,
		})...)
		go join(stream, destConn)
	}
}

// StartConnect creates a yamux client on the secure connection, listens on
// bindAddr, and for each local connection opens a new stream and joins.
// Blocks until secureConn is closed.
func StartConnect(secureConn net.Conn, bindAddr string) error {
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

	logger.Info("tunnel.connect listening", logger.Context("params", map[string]any{
		"bind_addr": bindAddr,
	})...)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			logger.Warn("tunnel.connect accept error", zap.Error(err))
			return err
		}

		stream, err := session.Open()
		if err != nil {
			logger.Warn("tunnel.connect open stream failed", zap.Error(err))
			localConn.Close()
			continue
		}

		logger.Debug("tunnel.connect new stream opened for local conn")
		go join(localConn, stream)
	}
}

// join performs bidirectional copy between two connections.
// Closes both when either side finishes.
func join(c1, c2 net.Conn) {
	defer c1.Close()
	defer c2.Close()

	done := make(chan struct{}, 1)
	go func() {
		io.Copy(c2, c1)
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
