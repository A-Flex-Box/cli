package wormhole

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
)

// roomSlot holds a waiting connection by role (0=sender, 1=receiver).
// Relay only pairs opposite roles to avoid PAKE "can't have its own role".
type roomSlot struct {
	waitingRole int
	ch          chan net.Conn
}

// RelayServer pairs connections by RoomID and role: sender only with receiver.
type RelayServer struct {
	timeout time.Duration
	mu      sync.Mutex
	rooms   map[string]*roomSlot
}

// NewRelayServer creates a relay server with the given pairing timeout.
func NewRelayServer(timeout time.Duration) *RelayServer {
	return &RelayServer{
		timeout: timeout,
		rooms:   make(map[string]*roomSlot),
	}
}

// HandleConn handles a single connection: read RoomID+role, match opposite role or wait, then pipe.
func (r *RelayServer) HandleConn(conn net.Conn) {
	closeOnReturn := true
	defer func() {
		if closeOnReturn {
			conn.Close()
		}
	}()

	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetNoDelay(true)
	}

	roomID := make([]byte, roomIDLen)
	if _, err := io.ReadFull(conn, roomID); err != nil {
		logger.Warn("relay.HandleConn read room_id failed", logger.Context("params", map[string]any{"remote": conn.RemoteAddr().String(), "error": err.Error()})...)
		return
	}
	roleBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, roleBuf); err != nil {
		logger.Warn("relay.HandleConn read role failed", logger.Context("params", map[string]any{"remote": conn.RemoteAddr().String(), "error": err.Error()})...)
		return
	}
	role := int(roleBuf[0])
	key := string(roomID)
	logger.Info("relay.HandleConn", logger.Context("params", map[string]any{
		"remote": conn.RemoteAddr().String(), "room_id": key, "role": role,
	})...)

	r.mu.Lock()
	slot, exists := r.rooms[key]
	if exists {
		if slot.waitingRole == role {
			// Same role (sender+sender or receiver+receiver) - reject to avoid PAKE error.
			r.mu.Unlock()
			logger.Warn("relay.HandleConn same role rejected", logger.Context("params", map[string]any{
				"remote": conn.RemoteAddr().String(), "room_id": key, "role": role,
			})...)
			return
		}
		// Opposite role: second arrival, match.
		delete(r.rooms, key)
		ch := slot.ch
		r.mu.Unlock()
		logger.Info("relay.HandleConn matched", logger.Context("params", map[string]any{"room_id": key, "remote": conn.RemoteAddr().String()})...)
		ch <- conn
		peer := <-ch
		close(ch)
		closeOnReturn = false
		_ = peer
		return
	}
	ch := make(chan net.Conn, 1)
	r.rooms[key] = &roomSlot{waitingRole: role, ch: ch}
	r.mu.Unlock()

	// First arrival: wait for opposite role.
	logger.Info("relay.HandleConn waiting for peer", logger.Context("params", map[string]any{
		"remote": conn.RemoteAddr().String(), "room_id": key, "role": role, "timeout_sec": r.timeout.Seconds(),
	})...)
	select {
	case peer := <-ch:
		ch <- conn
		closeOnReturn = false
		logger.Info("relay.HandleConn piping", logger.Context("params", map[string]any{"room_id": key})...)
		r.pipe(conn, peer)
	case <-time.After(r.timeout):
		r.mu.Lock()
		close(ch)
		delete(r.rooms, key)
		r.mu.Unlock()
		logger.Warn("relay.HandleConn pairing timeout", logger.Context("params", map[string]any{"remote": conn.RemoteAddr().String(), "room_id": key})...)
	}
}

func (r *RelayServer) pipe(a, b net.Conn) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	done := make(chan struct{}, 1)
	go func() {
		io.CopyBuffer(a, b, buf)
		if tcp, ok := a.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		io.CopyBuffer(b, a, buf)
		if tcp, ok := b.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
		done <- struct{}{}
	}()
	<-done
	a.Close()
	b.Close()
}
