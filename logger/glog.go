package logger

import (
	"github.com/golang/glog"
)

// GlogLogger implements the Logger interface for logging using the glog library with configurable call depth.
type GlogLogger struct {
	depth int
}

// Debug logs a debug-level message with the specified format and arguments.
func (logger *GlogLogger) Debugf(msg string, args ...any) {
	glog.InfoDepthf(logger.depth, msg, args...)
}

// Info logs an informational-level message with the specified format and optional arguments.
func (logger *GlogLogger) Infof(msg string, args ...any) {
	glog.InfoDepthf(logger.depth, msg, args...)
}

// Warn logs a warning-level message with the specified format and arguments.
func (logger *GlogLogger) Warnf(msg string, args ...any) {
	glog.WarningDepthf(logger.depth, msg, args...)
}

// Error logs an error-level message with the specified format and arguments.
func (logger *GlogLogger) Errorf(msg string, args ...any) {
	glog.ErrorDepthf(logger.depth, msg, args...)
}

// Fatal logs a fatal-level message with the specified format and arguments, then exits the application.
func (logger *GlogLogger) Fatalf(msg string, args ...any) {
	glog.FatalDepthf(logger.depth, msg, args...)
}

func NewGlogLogger() Logger {
	return &GlogLogger{
		depth: 1,
	}
}
