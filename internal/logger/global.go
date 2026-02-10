package logger

import (
	"sync"

	"go.uber.org/zap"
)

var (
	global     *zap.Logger
	globalOnce sync.Once
)

// SetGlobalLogger sets the global logger (e.g. from root PersistentPreRun).
func SetGlobalLogger(l *zap.Logger) {
	global = l
}

// Global returns the global logger. If not set, returns default NewLogger().
func Global() *zap.Logger {
	globalOnce.Do(func() {
		if global == nil {
			global = NewLogger()
		}
	})
	return global
}

// Package-level API: call directly without logger.Global().

func Debug(msg string, fields ...zap.Field)   { Global().Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)    { Global().Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)    { Global().Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field)   { Global().Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field)   { Global().Fatal(msg, fields...) }
func Sync() error                             { return Global().Sync() }
