package wormhole

import (
	"io"
	"net"
	"sync"
	"time"
)

// RelayServer is a dumb relay that pairs connections by RoomID.
type RelayServer struct {
	timeout time.Duration
	mu      sync.Mutex
	rooms   map[string]chan net.Conn
}

// NewRelayServer creates a relay server with the given pairing timeout.
func NewRelayServer(timeout time.Duration) *RelayServer {
	return &RelayServer{
		timeout: timeout,
		rooms:   make(map[string]chan net.Conn),
	}
}

// HandleConn handles a single connection: read RoomID, match or wait, then pipe.
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
		return
	}
	key := string(roomID)

	r.mu.Lock()
	ch, exists := r.rooms[key]
	if exists {
		delete(r.rooms, key)
		r.mu.Unlock()
		peer := <-ch
		close(ch)
		closeOnReturn = false
		r.pipe(conn, peer)
		return
	}
	ch = make(chan net.Conn, 1)
	r.rooms[key] = ch
	r.mu.Unlock()

	ch <- conn
	select {
	case peer := <-ch:
		closeOnReturn = false
		r.pipe(conn, peer)
	case <-time.After(r.timeout):
		r.mu.Lock()
		if c, ok := <-ch; ok {
			close(ch)
			c.Close()
		}
		delete(r.rooms, key)
		r.mu.Unlock()
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
