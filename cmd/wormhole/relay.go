package wormhole

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	wh "github.com/A-Flex-Box/cli/internal/wormhole"
	"github.com/spf13/cobra"
)

func newRelayCmd() *cobra.Command {
	var port int
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:     "relay",
		Short:   "Run the relay server (dumb TCP signal server)",
		Example: "cli wormhole relay -p 9000",
		Run: func(cmd *cobra.Command, args []string) {
			addr := fmt.Sprintf(":%d", port)
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				fmt.Printf("Failed to listen: %v\n", err)
				os.Exit(1)
			}
			defer ln.Close()

			srv := wh.NewRelayServer(timeout)
			logger.Info("relay.listening", logger.Context("params", map[string]any{
				"addr": addr, "timeout_sec": timeout.Seconds(),
			})...)
			fmt.Printf("Relay listening on %s (timeout: %v)\n", addr, timeout)

			for {
				conn, err := ln.Accept()
				if err != nil {
					logger.Warn("relay.Accept error", logger.Context("params", map[string]any{"error": err.Error()})...)
					fmt.Printf("Accept error: %v\n", err)
					continue
				}
				logger.Info("relay.Accept new conn", logger.Context("params", map[string]any{"remote": conn.RemoteAddr().String()})...)
				go srv.HandleConn(conn)
			}
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", envInt("CLI_RELAY_PORT", 9000), "Port to listen on")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", envDuration("CLI_RELAY_TIMEOUT", 60*time.Second), "Pairing wait timeout")
	return cmd
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
