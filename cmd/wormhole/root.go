package wormhole

import (
	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/spf13/cobra"
)

// NewCmd returns the wormhole command with relay, send, receive subcommands.
// Receives injected WormholeConfig.
func NewCmd(cfg *config.WormholeConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wormhole",
		Short:   "Secure P2P file and text transfer via relay",
		Example: "cli wormhole send file ./data.zip",
	}
	cmd.AddCommand(newRelayCmd(), newSendCmd(cfg), newReceiveCmd(cfg))
	return cmd
}
