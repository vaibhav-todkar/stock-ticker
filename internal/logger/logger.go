// Package logger provides a production-grade structured logger backed by Zap.
package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps a zap.Logger for structured, levelled logging.
type Logger struct {
	*zap.Logger
}

// New creates a new Logger with the given level and format ("json" or "console").
func New(level, format string) (*Logger, error) {
	zapLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	var cfg zap.Config
	if strings.ToLower(format) == "console" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := cfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &Logger{logger}, nil
}

// parseLevel converts a string log level to a zapcore.Level.
func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level %q, defaulting to info", level)
	}
}

// Sync flushes any buffered log entries. Should be called before program exit.
func (l *Logger) Sync() {
	_ = l.Logger.Sync()
}
