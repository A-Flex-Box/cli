package wormhole

import (
	"fmt"
	"net"
	"os"

	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/A-Flex-Box/cli/internal/logger"
	wh "github.com/A-Flex-Box/cli/internal/wormhole"
	"github.com/spf13/cobra"
)

func newConnectCmd(cfg *config.WormholeConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect <code> <local_bind_addr>",
		Short: "Map a remote service to a local address",
		Long:  "Connect to a remote exposed service using the pairing code. Local bind address can be e.g. :9090 or 127.0.0.1:9000.",
		Example: `  cli wormhole connect 7-magic-fish :9090
  cli wormhole connect abc1 :9000`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			relayAddr := cfg.GetActiveRelayAddr()
			if relayAddr == "" {
				fmt.Println("No active relay. Run: cli config use <name>")
				os.Exit(1)
			}

			code := args[0]
			bindAddr := args[1]
			if code == "" {
				fmt.Println("Code is required")
				os.Exit(1)
			}
			if _, _, err := net.SplitHostPort(bindAddr); err != nil {
				// ":9090" is valid for SplitHostPort (returns "", "9090", nil)
				fmt.Printf("Invalid bind address: %s (use e.g. :9090 or 127.0.0.1:9000)\n", bindAddr)
				os.Exit(1)
			}

			fmt.Printf("Listening on %s...\n", bindAddr)

			logger.Info("wormhole.connect start", logger.Context("params", map[string]any{
				"relay_addr": relayAddr, "code": code, "bind_addr": bindAddr,
			})...)

			if err := wh.ConnectTunnel(relayAddr, code, bindAddr); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	return cmd
}
