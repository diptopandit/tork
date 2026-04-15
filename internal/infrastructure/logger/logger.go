package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a minimal structured logging interface.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Sync() error
}

// ParseLevel converts a string level name to a zapcore.Level.
// Accepted values: "debug", "info", "warn", "error". Default: InfoLevel.
func ParseLevel(s string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zap.DebugLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

// New creates a Logger that writes to tork.log in the given data directory
// at the specified level.
func New(dataDir string, level zapcore.Level) (Logger, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("logger: create log dir: %w", err)
	}
	logPath := filepath.Join(dataDir, "tork.log")

	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "ts"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("logger: open log file: %w", err)
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.AddSync(file),
		level,
	)
	return zap.New(core), nil
}

// Nop returns a Logger that discards all output.
func Nop() Logger {
	return zap.NewNop()
}
