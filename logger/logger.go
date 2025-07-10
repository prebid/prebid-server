package logger

import "fmt"

var defaultDepth int = 1

var logger Logger = NewDefaultLogger(defaultDepth)

// LoggerType represents different logger implementations
type LoggerType string

const (
	LoggerTypeDefault LoggerType = "default"
	LoggerTypeCustom  LoggerType = "custom"
)

// LoggerConfig holds configuration for different logger types
type LoggerConfig struct {
	Type         LoggerType
	Depth        *int
	CustomLogger Logger // For custom logger instances
	CustomName   string // For registered custom loggers
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
	if config.Depth == nil {
		config.Depth = &defaultDepth
	}

	var newLogger Logger

	switch config.Type {
	case LoggerTypeDefault:
		newLogger = NewDefaultLogger(*config.Depth)
	case LoggerTypeCustom:
		if config.CustomLogger != nil {
			newLogger = config.CustomLogger
		} else if config.CustomName != "" {
			customLogger, err := GetLoggerByName(config.CustomName)
			if err != nil {
				return fmt.Errorf("failed to get custom logger '%s': %w", config.CustomName, err)
			}
			newLogger = customLogger
		} else {
			return fmt.Errorf("custom logger type requires either CustomLogger instance or CustomName")
		}
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

// SetCustomLoggerByName sets a custom logger by its registered name
func SetCustomLoggerByName(name string) error {
	customLogger, err := GetLoggerByName(name)
	if err != nil {
		return fmt.Errorf("failed to set custom logger '%s': %w", name, err)
	}
	logger = customLogger
	return nil
}

// NewCustom creates a new custom logger configuration
func NewCustom(customLogger Logger) error {
	config := &LoggerConfig{
		Type:         LoggerTypeCustom,
		CustomLogger: customLogger,
	}
	return NewWithConfig(config)
}

// NewCustomByName creates a new custom logger configuration using a registered logger name
func NewCustomByName(name string) error {
	config := &LoggerConfig{
		Type:       LoggerTypeCustom,
		CustomName: name,
	}
	return NewWithConfig(config)
}

// GetCurrentLogger returns the current logger instance
func GetCurrentLogger() Logger {
	return logger
}

func Info(args ...any) {
	logger.Info(args...)
}

func Infof(format string, args ...any) {
	logger.Infof(format, args)
}

func Warning(args ...any) {
	logger.Warning(args...)
}

func Warningf(format string, args ...any) {
	logger.Warningf(format, args)
}

func Error(args ...any) {
	logger.Error(args...)
}

func Errorf(format string, args ...any) {
	logger.Errorf(format, args)
}

func Exitf(format string, args ...any) {
	logger.Exitf(format, args)
}

func Fatal(args ...any) {
	logger.Fatal(args...)
}

func Fatalf(format string, args ...any) {
	logger.Fatalf(format, args)
}
