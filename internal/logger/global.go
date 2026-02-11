package logger

import (
	"go.uber.org/zap"
)

// Package-level API: use these instead of logger.L.Info(...) for convenience.
// L and S are set by Setup() in cmd/root.go PersistentPreRun.

// Debug logs at DEBUG level. Hidden on console when --debug is false.
func Debug(msg string, fields ...zap.Field) {
	if L != nil {
		L.Debug(msg, fields...)
	}
}

// Info logs at INFO level.
func Info(msg string, fields ...zap.Field) {
	if L != nil {
		L.Info(msg, fields...)
	}
}

// Warn logs at WARN level.
func Warn(msg string, fields ...zap.Field) {
	if L != nil {
		L.Warn(msg, fields...)
	}
}

// Error logs at ERROR level.
func Error(msg string, fields ...zap.Field) {
	if L != nil {
		L.Error(msg, fields...)
	}
}

// Fatal logs at FATAL level and exits.
func Fatal(msg string, fields ...zap.Field) {
	if L != nil {
		L.Fatal(msg, fields...)
	}
}

// Infof logs with fmt-style template (SugaredLogger).
func Infof(template string, args ...interface{}) {
	if S != nil {
		S.Infof(template, args...)
	}
}

// Sync flushes buffered log entries. Call in PersistentPostRun.
func Sync() error {
	if L != nil {
		return L.Sync()
	}
	return nil
}

// Global returns the global *zap.Logger. For code that needs to pass logger to functions
// (e.g. printerpkg.AutoSetup). Returns L; if L is nil (before Setup), returns zap.NewNop().
func Global() *zap.Logger {
	if L != nil {
		return L
	}
	return zap.NewNop()
}
