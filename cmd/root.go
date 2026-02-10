package cmd

import (
	"fmt"
	"os"
	"runtime"

	cmdarchive "github.com/A-Flex-Box/cli/cmd/archive"
	cmdai "github.com/A-Flex-Box/cli/cmd/ai"
	cmdconfig "github.com/A-Flex-Box/cli/cmd/config"
	cmddoctor "github.com/A-Flex-Box/cli/cmd/doctor"
	cmdhistory "github.com/A-Flex-Box/cli/cmd/history"
	cmdprinter "github.com/A-Flex-Box/cli/cmd/printer"
	cmdprompt "github.com/A-Flex-Box/cli/cmd/prompt"
	cmdvalidate "github.com/A-Flex-Box/cli/cmd/validate"
	cmdwormhole "github.com/A-Flex-Box/cli/cmd/wormhole"
	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:     "cli",
	Short:   "Go CLI Tool",
	Long:    `A powerful CLI tool created via automated scaffolding.`,
	Example: "cli version",
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print build info",
	Example: "cli version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cli Build Info:\n")
		fmt.Printf(" Version: %s\n", version)
		fmt.Printf(" Commit:  %s\n", commit)
		fmt.Printf(" Date:    %s\n", date)
		fmt.Printf(" Go:      %s\n", runtime.Version())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)

	mgr := config.NewManager()
	cfg, err := mgr.Load()
	if err != nil {
		cfg = &config.Root{
			Wormhole: config.WormholeConfig{
				ActiveRelay: "public",
				Relays: map[string]string{
					"public": "tcp://relay.flex-box.dev:9000",
					"local":  "tcp://127.0.0.1:9000",
				},
			},
		}
	}

	rootCmd.AddCommand(cmdai.NewCmd())
	rootCmd.AddCommand(cmdarchive.NewCmd())
	rootCmd.AddCommand(cmdconfig.NewCmd(cfg, mgr))
	rootCmd.AddCommand(cmddoctor.NewCmd())
	rootCmd.AddCommand(cmdhistory.NewCmd())
	rootCmd.AddCommand(cmdprinter.NewCmd())
	rootCmd.AddCommand(cmdprompt.NewCmd())
	rootCmd.AddCommand(cmdvalidate.NewCmd())
	rootCmd.AddCommand(cmdwormhole.NewCmd(&cfg.Wormhole))
}
