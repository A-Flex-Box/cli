package wormhole

import (
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/A-Flex-Box/cli/internal/logger"
	wh "github.com/A-Flex-Box/cli/internal/wormhole"
	"github.com/spf13/cobra"
)

func newReceiveCmd(cfg *config.WormholeConfig) *cobra.Command {
	var code, outDir string

	cmd := &cobra.Command{
		Use:     "receive",
		Short:   "Receive file or text",
		Example: "cli wormhole receive -c abc-123",
		Run: func(cmd *cobra.Command, args []string) {
			relayAddr := cfg.GetActiveRelayAddr()
			logger.Info("wormhole.receive cmd start", logger.Context("params", map[string]any{
				"relay_addr": relayAddr, "active_relay": cfg.ActiveRelay, "code": code, "out_dir": outDir,
			})...)
			if relayAddr == "" {
				fmt.Println("No active relay. Run: cli config use <name>")
				os.Exit(1)
			}

			pairCode := code
			if pairCode == "" {
				fmt.Print("Enter code from sender: ")
				fmt.Scanln(&pairCode)
				if pairCode == "" {
					fmt.Println("Code required")
					os.Exit(1)
				}
			}

			dir := outDir
			if dir == "" {
				dir = "."
			}

			var receivedText string
			var result wh.ReceiveResult
			err := wh.RunTransferUI("Receiving...", 0, pairCode, &result, func(onProgress func(int64, int64)) error {
				return wh.Receive(relayAddr, pairCode, dir, onProgress, &receivedText, &result)
			})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&code, "code", "c", "", "Pairing code from sender")
	cmd.Flags().StringVarP(&outDir, "out", "o", ".", "Output directory for received files")
	return cmd
}
