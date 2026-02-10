package archiver

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/A-Flex-Box/cli/internal/logger"
	"go.uber.org/zap"
)

type ArchiveConfig struct {
	DeleteSource bool
}

type Manager struct {
	cfg ArchiveConfig
}

func NewManager(cfg ArchiveConfig) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) Run() error {
	timestamp := time.Now().Format("20060102_150405")
	archiveName := fmt.Sprintf("archive_%s.tar.gz", timestamp)
	
	// 正则: 保留 archive_YYYYMMDD_HHMMSS.tar.gz 格式的历史归档
	validArchiveRegex := regexp.MustCompile(`^archive_\d{8}_\d{6}\.tar\.gz$`)

	logger.Info("archive start", zap.String("file", archiveName))

	outFile, err := os.Create(archiveName)
	if err != nil { return fmt.Errorf("create file err: %w", err) }
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	var filesToDelete []string
	baseDir, _ := os.Getwd()
	exePath, _ := os.Executable()
	exeName := filepath.Base(exePath)

	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil { return err }
		if relPath == "." { return nil }

		// 排除自身生成的归档、Git目录、CLI本身、History目录
		if relPath == archiveName { return nil }
		if info.Name() == ".git" || relPath == ".git" { return filepath.SkipDir }
		if info.Name() == "history" || relPath == "history" { return filepath.SkipDir }
		if info.Name() == exeName { return nil }

		// 保留历史标准归档
		if validArchiveRegex.MatchString(info.Name()) {
			logger.Info("archive skipping historical", zap.String("file", relPath))
			return nil 
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil { return err }
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil { return err }

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil { return err }
			defer file.Close()
			if _, err := io.Copy(tw, file); err != nil { return err }
			filesToDelete = append(filesToDelete, path)
		}
		return nil
	})

	if err != nil { return err }

	// Ensure flush
	tw.Close(); gw.Close(); outFile.Close()
	logger.Info("archive created", zap.String("path", archiveName))

	if m.cfg.DeleteSource {
		logger.Info("archive deleting source files")
		for _, f := range filesToDelete {
			os.Remove(f)
		}
	}
	return nil
}
