package logger

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents log level: debug, info, warn, error.
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// ParseLogLevel parses string to LogLevel (case-insensitive). Defaults to info.
func ParseLogLevel(s string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug", "d":
		return LevelDebug
	case "warn", "warning", "w":
		return LevelWarn
	case "error", "err", "e":
		return LevelError
	default:
		return LevelInfo
	}
}

// LogRotation holds lumberjack rotation parameters.
type LogRotation struct {
	MaxSize    int  // MB
	MaxBackups int
	MaxAge     int  // days
	Compress   bool
}

// LoggerConfig controls logger behavior.
type LoggerConfig struct {
	Level       LogLevel   // debug, info, warn, error
	LogPath     string     // empty: console only; non-empty: tee to file
	LogRotation LogRotation
}

// DefaultLogRotation returns default rotation params.
func DefaultLogRotation() LogRotation {
	return LogRotation{
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
	}
}

// toZapLevel converts LogLevel to zapcore.Level.
func toZapLevel(l LogLevel) zapcore.Level {
	switch l {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// NewLoggerWithConfig creates a logger from config.
func NewLoggerWithConfig(cfg LoggerConfig) *zap.Logger {
	level := toZapLevel(cfg.Level)
	if level == 0 {
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	cores := []zapcore.Core{consoleCore}

	if cfg.LogPath != "" {
		rot := cfg.LogRotation
		if rot.MaxSize == 0 {
			rot = DefaultLogRotation()
		}
		writer := &lumberjack.Logger{
			Filename:   cfg.LogPath,
			MaxSize:    rot.MaxSize,
			MaxBackups: rot.MaxBackups,
			MaxAge:     rot.MaxAge,
			Compress:   rot.Compress,
		}
		fileEncoderConfig := zap.NewProductionEncoderConfig()
		fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(fileEncoderConfig),
			zapcore.AddSync(writer),
			level,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	return zap.New(core)
}

// DefaultLogPath returns os.TempDir()/cli_YYYYMMDD_HHMMSS.log.
func DefaultLogPath() string {
	return filepath.Join(os.TempDir(), "cli_"+time.Now().Format("20060102_150405")+".log")
}

// NewLogger returns a logger with default config (InfoLevel, stdout only).
func NewLogger() *zap.Logger {
	return NewLoggerWithConfig(LoggerConfig{Level: LevelInfo})
}

// Context builds zap fields from a map for readable structured logging.
// Usage: log.Info("operation", logger.Context("params", map[string]any{"a": 1})...)
func Context(blockName string, data map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(data)+1)
	fields = append(fields, zap.String("block", blockName))
	for k, v := range data {
		fields = append(fields, zap.Any(k, v))
	}
	return fields
}

// WithContext returns a logger with op and params attached for structured logging.
func WithContext(log *zap.Logger, op string, params map[string]any) *zap.Logger {
	fs := make([]zap.Field, 0, len(params)+1)
	fs = append(fs, zap.String("op", op))
	for k, v := range params {
		fs = append(fs, zap.Any(k, v))
	}
	return log.With(fs...)
}
