package wormhole

import (
	"fmt"
	"os"
	"strconv"

	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/A-Flex-Box/cli/internal/logger"
	wh "github.com/A-Flex-Box/cli/internal/wormhole"
	"github.com/spf13/cobra"
)

func newExposeCmd(cfg *config.WormholeConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "expose <local_port>",
		Short: "Share a local service via the wormhole tunnel",
		Long:  "Expose a local port (e.g. 8080) through the wormhole. Remote users with the code can connect via wormhole connect.",
		Example: `  cli wormhole expose 8080
  cli wormhole expose 3000`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			relayAddr := cfg.GetActiveRelayAddr()
			if relayAddr == "" {
				fmt.Println("No active relay. Run: cli config use <name>")
				os.Exit(1)
			}

			portStr := args[0]
			if _, err := strconv.Atoi(portStr); err != nil {
				fmt.Printf("Invalid port: %s\n", portStr)
				os.Exit(1)
			}

			code := wh.GenerateCode()
			fmt.Printf("Code is: %s\n", code)
			fmt.Println("Tunnel established. Waiting for connections...")

			logger.Info("wormhole.expose start", logger.Context("params", map[string]any{
				"relay_addr": relayAddr, "code": code, "local_port": portStr,
			})...)

			if err := wh.ExposeTunnel(relayAddr, code, portStr); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	return cmd
}
