package cmd

import (
	"fmt"
	"os"
	"runtime"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "Go CLI Tool",
	Long:  `A powerful CLI tool created via automated scaffolding.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print build info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cli Build Info:\n")
		fmt.Printf(" Version: %s\n", version)
		fmt.Printf(" Commit:  %s\n", commit)
		fmt.Printf(" Date:    %s\n", date)
		fmt.Printf(" Go:      %s\n", runtime.Version())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil { os.Exit(1) }
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
