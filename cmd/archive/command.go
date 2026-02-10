package archive

import (
	"github.com/A-Flex-Box/cli/internal/archiver"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewCmd returns the archive command.
func NewCmd() *cobra.Command {
	var deleteFiles bool

	cmd := &cobra.Command{
		Use:     "archive",
		Short:   "Create tar.gz archive",
		Example: "cli archive -d",
		Run: func(cmd *cobra.Command, args []string) {
			defer logger.Sync()
			logger.Info("archive cmd start", logger.Context("params", map[string]any{
				"delete_source": deleteFiles,
			})...)
			cfg := archiver.ArchiveConfig{DeleteSource: deleteFiles}
			if err := archiver.NewManager(cfg).Run(); err != nil {
				logger.Fatal("Archive failed", zap.Error(err))
			}
		},
	}
	cmd.Flags().BoolVarP(&deleteFiles, "delete", "d", false, "Delete source files after archive")
	return cmd
}
