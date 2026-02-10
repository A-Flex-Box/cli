package wormhole

import "net"

// UpgradeConn runs PAKE handshake and returns a secured connection.
// Convenience wrapper around UpgradeDefault for client use.
func UpgradeConn(conn net.Conn, password string, isSender bool) (*SecureConn, error) {
	return UpgradeDefault(conn, password, isSender)
}
