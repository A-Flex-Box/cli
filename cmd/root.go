package cmd

import (
	"fmt"
	"os"
	"runtime"

	cmdarchive "github.com/A-Flex-Box/cli/cmd/archive"
	cmdconfig "github.com/A-Flex-Box/cli/cmd/config"
	cmddoctor "github.com/A-Flex-Box/cli/cmd/doctor"
	cmdhistory "github.com/A-Flex-Box/cli/cmd/history"
	cmdprinter "github.com/A-Flex-Box/cli/cmd/printer"
	cmdwormhole "github.com/A-Flex-Box/cli/cmd/wormhole"
	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "none"
	date      = "unknown"
	debugMode bool
)

var rootCmd = &cobra.Command{
	Use:     "cli",
	Short:   "Go CLI Tool",
	Long:    `A powerful CLI tool created via automated scaffolding.`,
	Example: "cli version",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// --debug controls console verbosity: false=INFO+, true=DEBUG+
		// Config can override: if config has Debug or log_level=debug, enable debug
		debug := debugMode
		if !debug {
			mgr := config.NewManager()
			cfg, _ := mgr.Load()
			if cfg != nil && (cfg.Debug || cfg.LogLevel == "debug") {
				debug = true
			}
		}
		logger.Setup(debug)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		_ = logger.Sync()
	},
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
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logs on console (file always records debug)")
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

	rootCmd.AddCommand(cmdarchive.NewCmd())
	rootCmd.AddCommand(cmdconfig.NewCmd(cfg, mgr))
	rootCmd.AddCommand(cmddoctor.NewCmd())
	rootCmd.AddCommand(cmdhistory.NewCmd())
	rootCmd.AddCommand(cmdprinter.NewCmd())
	rootCmd.AddCommand(cmdwormhole.NewCmd(&cfg.Wormhole))
}
