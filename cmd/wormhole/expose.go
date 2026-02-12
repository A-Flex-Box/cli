package wormhole

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/A-Flex-Box/cli/internal/logger"
	wh "github.com/A-Flex-Box/cli/internal/wormhole"
	"github.com/spf13/cobra"
)

func newExposeCmd(cfg *config.WormholeConfig) *cobra.Command {
	var code string

	cmd := &cobra.Command{
		Use:   "expose <local_port>",
		Short: "Share a local service via the wormhole tunnel",
		Long:  "Expose a local port (e.g. 8080) through the wormhole. Remote users with the code can connect via wormhole connect.",
		Example: `  cli wormhole expose 8080
  cli wormhole expose -c abc1 8080
  cli wormhole expose 3000`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			relayAddr := cfg.GetActiveRelayAddr()
			if relayAddr == "" {
				fmt.Println("No active relay. Run: cli config use <name>")
				os.Exit(1)
			}

			portStr := strings.TrimPrefix(args[0], ":")
			if _, err := strconv.Atoi(portStr); err != nil {
				fmt.Printf("Invalid port: %s\n", args[0])
				os.Exit(1)
			}

			pairCode := code
			if pairCode == "" {
				pairCode = wh.GenerateCode()
			}

			logger.Info("wormhole.expose start", logger.Context("params", map[string]any{
				"relay_addr": relayAddr, "code": pairCode, "local_port": portStr,
			})...)

			if err := wh.RunTunnelUI("expose", pairCode, portStr, func(opts *wh.TunnelOptions) error {
				return wh.ExposeTunnel(relayAddr, pairCode, portStr, opts)
			}); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&code, "code", "c", "", "Pairing code (generated if empty)")
	return cmd
}
