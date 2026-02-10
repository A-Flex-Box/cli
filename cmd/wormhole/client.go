package wormhole

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/A-Flex-Box/cli/internal/config"
	wh "github.com/A-Flex-Box/cli/internal/wormhole"
	"github.com/spf13/cobra"
)

func newSendCmd(cfg *config.WormholeConfig) *cobra.Command {
	var code string

	cmd := &cobra.Command{
		Use:     "send [file|text] [path|content]",
		Short:   "Send file or text",
		Long:    "wormhole send file <path>  - send a file\nwormhole send text <content> - send text",
		Example: "cli wormhole send file ./report.pdf\n  cli wormhole send text 'Hello'",
		Args:    cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			relayAddr := cfg.GetActiveRelayAddr()
			if relayAddr == "" {
				fmt.Println("No active relay. Run: cli config use <name>")
				os.Exit(1)
			}

			pairCode := code
			if pairCode == "" {
				pairCode = wh.GenerateCode()
				fmt.Printf("Your code: %s (share with receiver)\n", pairCode)
			}

			mode := args[0]
			switch mode {
			case "file":
				filePath := args[1]
				info, err := os.Stat(filePath)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				title := "Sending: " + filepath.Base(filePath)
				err = wh.RunTransferUI(title, info.Size(), func(onProgress func(int64, int64)) error {
					return wh.SendFile(relayAddr, pairCode, filePath, onProgress)
				})
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
			case "text":
				text := args[1]
				if err := wh.SendText(relayAddr, pairCode, text); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(wh.RenderSecureBox())
			default:
				fmt.Printf("Unknown mode: %s (use file or text)\n", mode)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&code, "code", "c", "", "Pairing code (generated if empty)")
	return cmd
}
