package logger

import (
	"context"
	"fmt"
)

var defaultDepth int = 1

var logger Logger = NewSlogLogger()

// LoggerType represents different logger implementations
type LoggerType string

const (
	LoggerTypeSlog   LoggerType = "slog"
	LoggerTypeGlog   LoggerType = "glog"
	LoggerTypeCustom LoggerType = "custom"
)

// LoggerConfig holds configuration for different logger types
type LoggerConfig struct {
	Type  LoggerType
	Depth *int

	CustomLogger Logger // For custom logger instances
}

// New initializes a logger configuration with the specified type and depth, and applies it using NewWithConfig.
func New(loggerType string, depth *int) error {
	config := &LoggerConfig{
		Type:  LoggerType(loggerType),
		Depth: depth,
	}

	return NewWithConfig(config)
}

// NewWithConfig initializes a logger based on the provided LoggerConfig and sets it as the active logger.
// Returns an error if the configuration is invalid or logger initialization fails.
func NewWithConfig(config *LoggerConfig) error {
	if config == nil {
		return fmt.Errorf("logger config is nil")
	}

	if config.Depth == nil {
		config.Depth = &defaultDepth
	}

	var newLogger Logger

	switch config.Type {
	case LoggerTypeGlog:
		newLogger = NewGlogLogger(*config.Depth)

	case LoggerTypeSlog:
		newLogger = NewSlogLogger()

	case LoggerTypeCustom:
		if config.CustomLogger == nil {
			return fmt.Errorf("custom logger type requires CustomLogger instance")
		}

		newLogger = config.CustomLogger

	default:
		return fmt.Errorf("unsupported logger type: %s", config.Type)
	}

	logger = newLogger
	return nil

}

// SetCustomLogger directly sets a custom logger instance
func SetCustomLogger(customLogger Logger) {
	logger = customLogger
}

// GetCurrentLogger returns the current logger instance
func GetCurrentLogger() Logger {
	return logger
}

// Debug level logging
func Debug(msg any, args ...any) {
	logger.Debug(msg, args...)
}

func DebugContext(ctx context.Context, msg any, args ...any) {
	logger.DebugContext(ctx, msg, args...)
}

// Info level logging
func Info(msg any, args ...any) {
	logger.Info(msg, args...)
}

func InfoContext(ctx context.Context, msg any, args ...any) {
	logger.InfoContext(ctx, msg, args...)
}

// Warn level logging
func Warn(msg any, args ...any) {
	logger.Warn(msg, args...)
}

func WarnContext(ctx context.Context, msg any, args ...any) {
	logger.WarnContext(ctx, msg, args...)
}

// Error level logging
func Error(msg any, args ...any) {
	logger.Error(msg, args...)
}

func ErrorContext(ctx context.Context, msg any, args ...any) {
	logger.ErrorContext(ctx, msg, args...)
}
