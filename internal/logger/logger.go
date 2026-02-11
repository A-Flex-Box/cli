package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// =============================================================================
// Developer Note: Debug Logs & Verbosity
// =============================================================================
// Use logger.Debug() for verbose internal logic (e.g. PAKE handshake steps,
// connection establishment details, inferred metadata). These logs are
// HIDDEN by default when --debug is false. Only INFO and above appear on
// console. File output always records DEBUG and above for troubleshooting.
// =============================================================================

// ANSI color codes for Gin-style console output.
const (
	colorReset   = "\033[0m"
	colorMagenta = "\033[35m" // DEBUG
	colorGreen   = "\033[32m" // INFO
	colorYellow  = "\033[33m" // WARN
	colorRed     = "\033[31m" // ERROR
)

// Default file log path for flex-cli.
const defaultLogPath = "flex-cli.log"

// ginStyleLevelString returns colored [LEVEL] string.
func ginStyleLevelString(l zapcore.Level) string {
	var color, level string
	switch l {
	case zapcore.DebugLevel:
		color, level = colorMagenta, "DEBUG"
	case zapcore.InfoLevel:
		color, level = colorGreen, "INFO"
	case zapcore.WarnLevel:
		color, level = colorYellow, "WARN"
	case zapcore.ErrorLevel:
		color, level = colorRed, "ERROR"
	default:
		color, level = colorReset, l.CapitalString()
	}
	return color + "[" + level + "]" + colorReset
}

// ginStyleConsoleEncoder outputs: [LEVEL] TIME | CALLER | MESSAGE
type ginStyleConsoleEncoder struct {
	cfg zapcore.EncoderConfig
}

func newGinStyleConsoleEncoder(cfg zapcore.EncoderConfig) *ginStyleConsoleEncoder {
	return &ginStyleConsoleEncoder{cfg: cfg}
}

func (e *ginStyleConsoleEncoder) Clone() zapcore.Encoder {
	return &ginStyleConsoleEncoder{cfg: e.cfg}
}

func (e *ginStyleConsoleEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := buffer.NewPool().Get()

	line.AppendString(ginStyleLevelString(ent.Level))
	line.AppendString(" ")
	line.AppendString(ent.Time.Format("15:04:05"))
	line.AppendString(" | ")
	if ent.Caller.Defined {
		line.AppendString(ent.Caller.TrimmedPath())
	} else {
		line.AppendString("???")
	}
	line.AppendString(" | ")
	line.AppendString(ent.Message)

	// Append fields as key=value
	for _, f := range fields {
		line.AppendString("\t")
		line.AppendString(f.Key)
		line.AppendString("=")
		line.AppendString(fieldValueString(f))
	}
	line.AppendString("\n")
	return line, nil
}

// fieldValueString converts a zap Field to a string for console output.
func fieldValueString(f zapcore.Field) string {
	c := &valueCapture{}
	f.AddTo(c)
	return c.str
}

// valueCapture implements ObjectEncoder to capture a single field value as string.
type valueCapture struct {
	str string
}

func (c *valueCapture) AddArray(key string, v zapcore.ArrayMarshaler) error {
	arr := &sliceCapture{}
	_ = v.MarshalLogArray(arr)
	c.str = arr.string()
	return nil
}
func (c *valueCapture) AddObject(key string, v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	_ = v.MarshalLogObject(m)
	b, _ := json.Marshal(m.Fields)
	c.str = string(b)
	return nil
}
func (c *valueCapture) AddBinary(key string, val []byte)   { c.str = string(val) }
func (c *valueCapture) AddByteString(key string, val []byte) { c.str = string(val) }
func (c *valueCapture) AddBool(key string, val bool)       { c.str = fmt.Sprintf("%t", val) }
func (c *valueCapture) AddComplex128(key string, val complex128) { c.str = fmt.Sprintf("%v", val) }
func (c *valueCapture) AddComplex64(key string, val complex64)   { c.str = fmt.Sprintf("%v", val) }
func (c *valueCapture) AddDuration(key string, val time.Duration) { c.str = val.String() }
func (c *valueCapture) AddFloat64(key string, val float64) { c.str = fmt.Sprintf("%v", val) }
func (c *valueCapture) AddFloat32(key string, val float32) { c.str = fmt.Sprintf("%v", val) }
func (c *valueCapture) AddInt(key string, val int)         { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddInt64(key string, val int64)     { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddInt32(key string, val int32)     { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddInt16(key string, val int16)     { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddInt8(key string, val int8)       { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddString(key string, val string)   { c.str = val }
func (c *valueCapture) AddTime(key string, val time.Time) { c.str = val.Format(time.RFC3339) }
func (c *valueCapture) AddUint(key string, val uint)       { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddUint64(key string, val uint64)   { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddUint32(key string, val uint32)   { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddUint16(key string, val uint16)   { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddUint8(key string, val uint8)     { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddUintptr(key string, val uintptr) { c.str = fmt.Sprintf("%d", val) }
func (c *valueCapture) AddReflected(key string, val interface{}) error {
	b, _ := json.Marshal(val)
	c.str = string(b)
	if len(c.str) > 1 && c.str[0] == '"' && c.str[len(c.str)-1] == '"' {
		c.str = c.str[1 : len(c.str)-1]
	}
	return nil
}
func (c *valueCapture) OpenNamespace(key string) {}

type sliceCapture struct {
	elems []interface{}
}

func (s *sliceCapture) string() string { b, _ := json.Marshal(s.elems); return string(b) }
func (s *sliceCapture) AppendArray(v zapcore.ArrayMarshaler) error {
	sub := &sliceCapture{}
	_ = v.MarshalLogArray(sub)
	s.elems = append(s.elems, sub.elems)
	return nil
}
func (s *sliceCapture) AppendObject(v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	_ = v.MarshalLogObject(m)
	s.elems = append(s.elems, m.Fields)
	return nil
}
func (s *sliceCapture) AppendReflected(v interface{}) error { s.elems = append(s.elems, v); return nil }
func (s *sliceCapture) AppendBool(v bool)                   { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendByteString(v []byte)            { s.elems = append(s.elems, string(v)) }
func (s *sliceCapture) AppendComplex128(v complex128)        { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendComplex64(v complex64)          { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendDuration(v time.Duration)       { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendFloat64(v float64)              { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendFloat32(v float32)              { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendInt(v int)                      { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendInt64(v int64)                 { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendInt32(v int32)                 { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendInt16(v int16)                 { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendInt8(v int8)                   { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendString(v string)               { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendTime(v time.Time)               { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendUint(v uint)                   { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendUint64(v uint64)               { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendUint32(v uint32)               { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendUint16(v uint16)               { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendUint8(v uint8)                 { s.elems = append(s.elems, v) }
func (s *sliceCapture) AppendUintptr(v uintptr)             { s.elems = append(s.elems, v) }

// ObjectEncoder implementation - needed for zap Encoder interface.
func (e *ginStyleConsoleEncoder) AddArray(key string, v zapcore.ArrayMarshaler) error   { return nil }
func (e *ginStyleConsoleEncoder) AddObject(key string, v zapcore.ObjectMarshaler) error { return nil }
func (e *ginStyleConsoleEncoder) AddBinary(key string, val []byte)                      {}
func (e *ginStyleConsoleEncoder) AddByteString(key string, val []byte)                 {}
func (e *ginStyleConsoleEncoder) AddBool(key string, val bool)                          {}
func (e *ginStyleConsoleEncoder) AddComplex128(key string, val complex128)               {}
func (e *ginStyleConsoleEncoder) AddComplex64(key string, val complex64)                 {}
func (e *ginStyleConsoleEncoder) AddDuration(key string, val time.Duration)             {}
func (e *ginStyleConsoleEncoder) AddFloat64(key string, val float64)                    {}
func (e *ginStyleConsoleEncoder) AddFloat32(key string, val float32)                    {}
func (e *ginStyleConsoleEncoder) AddInt(key string, val int)                           {}
func (e *ginStyleConsoleEncoder) AddInt64(key string, val int64)                        {}
func (e *ginStyleConsoleEncoder) AddInt32(key string, val int32)                        {}
func (e *ginStyleConsoleEncoder) AddInt16(key string, val int16)                        {}
func (e *ginStyleConsoleEncoder) AddInt8(key string, val int8)                         {}
func (e *ginStyleConsoleEncoder) AddString(key string, val string)                     {}
func (e *ginStyleConsoleEncoder) AddTime(key string, val time.Time)                     {}
func (e *ginStyleConsoleEncoder) AddUint(key string, val uint)                          {}
func (e *ginStyleConsoleEncoder) AddUint64(key string, val uint64)                      {}
func (e *ginStyleConsoleEncoder) AddUint32(key string, val uint32)                      {}
func (e *ginStyleConsoleEncoder) AddUint16(key string, val uint16)                      {}
func (e *ginStyleConsoleEncoder) AddUint8(key string, val uint8)                        {}
func (e *ginStyleConsoleEncoder) AddUintptr(key string, val uintptr)                    {}
func (e *ginStyleConsoleEncoder) AddReflected(key string, val interface{}) error        { return nil }
func (e *ginStyleConsoleEncoder) OpenNamespace(key string)                              {}

// LogLevel represents log level.
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// ParseLogLevel parses string to LogLevel.
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

// DefaultLogPath returns /tmp/flex-cli.log or similar.
func DefaultLogPath() string {
	return filepath.Join(os.TempDir(), defaultLogPath)
}

// Context builds zap fields from a map.
func Context(blockName string, data map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(data)+1)
	fields = append(fields, zap.String("block", blockName))
	for k, v := range data {
		fields = append(fields, zap.Any(k, v))
	}
	return fields
}

// WithContext returns a logger with op and params attached.
func WithContext(log *zap.Logger, op string, params map[string]any) *zap.Logger {
	fs := make([]zap.Field, 0, len(params)+1)
	fs = append(fs, zap.String("op", op))
	for k, v := range params {
		fs = append(fs, zap.Any(k, v))
	}
	return log.With(fs...)
}

// L is the global zap.Logger. S is the global zap.SugaredLogger.
// They are set by Setup() and must be initialized before use.
var (
	L *zap.Logger
	S *zap.SugaredLogger
)

// Setup initializes the global logger with dual output:
// - Console: Gin-style, colored, concise. debug=false: WARN+ (quiet); debug=true: DEBUG+.
// - File: JSON, ISO8601, always DEBUG+. Rotated via lumberjack.
func Setup(debug bool) {
	consoleLevel := zapcore.WarnLevel
	if debug {
		consoleLevel = zapcore.DebugLevel
	}

	// Console encoder: [LEVEL] TIME | CALLER | MESSAGE
	consoleCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		CallerKey:      "caller",
		EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // overridden by gin encoder
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}
	consoleEnc := newGinStyleConsoleEncoder(consoleCfg)
	consoleCore := zapcore.NewCore(
		consoleEnc,
		zapcore.AddSync(os.Stdout),
		consoleLevel,
	)

	// File encoder: JSON, ISO8601, always DEBUG
	fileWriter := &lumberjack.Logger{
		Filename:   DefaultLogPath(),
		MaxSize:    10,  // MB
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}
	fileCfg := zap.NewProductionEncoderConfig()
	fileCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(fileCfg),
		zapcore.AddSync(fileWriter),
		zapcore.DebugLevel,
	)

	core := zapcore.NewTee(consoleCore, fileCore)
	L = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	S = L.Sugar()
}
