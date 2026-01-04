package cmd

import (
	"github.com/A-Flex-Box/cli/internal/archiver"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var deleteFiles bool

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Create tar.gz archive",
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()
		defer log.Sync()
		cfg := archiver.ArchiveConfig{DeleteSource: deleteFiles, Logger: log}
		if err := archiver.NewManager(cfg).Run(); err != nil {
			log.Fatal("Archive failed", zap.Error(err))
		}
	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	archiveCmd.Flags().BoolVarP(&deleteFiles, "delete", "d", false, "Delete source files after archive")
}
