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
	var code string

	cmd := &cobra.Command{
		Use:   "connect [code] <local_bind_addr>",
		Short: "Map a remote service to a local address",
		Long:  "Connect to a remote exposed service using the pairing code. Use -c to pass code, or provide code as first arg. Local bind address can be e.g. :9090 or 127.0.0.1:9000.",
		Example: `  cli wormhole connect abc1 :9090
  cli wormhole connect -c abc1 :9090
  cli wormhole connect -c 7-magic-fish :9000`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			relayAddr := cfg.GetActiveRelayAddr()
			if relayAddr == "" {
				fmt.Println("No active relay. Run: cli config use <name>")
				os.Exit(1)
			}

			var pairCode, bindAddr string
			if code != "" {
				// -c provided: args[0] is bindAddr
				pairCode = code
				bindAddr = args[0]
			} else if len(args) >= 2 {
				// no -c: args[0] is code, args[1] is bindAddr
				pairCode = args[0]
				bindAddr = args[1]
			} else {
				fmt.Println("Code required. Use -c <code> or provide as first argument.")
				os.Exit(1)
			}

			if pairCode == "" {
				fmt.Println("Code is required")
				os.Exit(1)
			}
			if _, _, err := net.SplitHostPort(bindAddr); err != nil {
				fmt.Printf("Invalid bind address: %s (use e.g. :9090 or 127.0.0.1:9000)\n", bindAddr)
				os.Exit(1)
			}

			logger.Info("wormhole.connect start", logger.Context("params", map[string]any{
				"relay_addr": relayAddr, "code": pairCode, "bind_addr": bindAddr,
			})...)

			if err := wh.RunTunnelUI("connect", pairCode, bindAddr, func(opts *wh.TunnelOptions) error {
				return wh.ConnectTunnel(relayAddr, pairCode, bindAddr, opts)
			}); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&code, "code", "c", "", "Pairing code from expose side")
	return cmd
}
