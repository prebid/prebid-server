package logger

import (
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGlogLogger(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("stderrthreshold", "INFO")

	logger := NewGlogLogger()

	assert.NotNil(t, logger, "NewGlogLogger should return a non-nil logger")

	glogLogger, ok := logger.(*GlogLogger)
	assert.True(t, ok, "Logger should be of type *GlogLogger")
	assert.Equal(t, 1, glogLogger.depth, "Default depth should be 1")
	assert.NotNil(t, glogLogger.slogLogger, "slogLogger field should be initialized")
}

func TestGlogLogger_ImplementsLoggerInterface(t *testing.T) {
	var _ Logger = (*GlogLogger)(nil)
}

func TestGlogLogger_Debug(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	logger := NewGlogLogger()

	// This test verifies the method can be called without panicking
	// Actual log output verification would require capturing stderr
	assert.NotPanics(t, func() {
		logger.Debugf("debug message")
	}, "Debug should not panic")

	assert.NotPanics(t, func() {
		logger.Debugf("debug message with args: %s, %d", "test", 123)
	}, "Debug with args should not panic")
}

func TestGlogLogger_Info(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	assert.NotPanics(t, func() {
		logger.Infof("info message")
	}, "Info should not panic")

	assert.NotPanics(t, func() {
		logger.Infof("info message with args: %s, %d", "test", 456)
	}, "Info with args should not panic")
}

func TestGlogLogger_Warn(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	assert.NotPanics(t, func() {
		logger.Warnf("warning message")
	}, "Warn should not panic")

	assert.NotPanics(t, func() {
		logger.Warnf("warning message with args: %s, %d", "test", 789)
	}, "Warn with args should not panic")
}

func TestGlogLogger_Error(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	assert.NotPanics(t, func() {
		logger.Errorf("error message")
	}, "Error should not panic")

	assert.NotPanics(t, func() {
		logger.Errorf("error message with args: %s, %d", "test", 999)
	}, "Error with args should not panic")
}

func TestGlogLogger_AllLevels(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	logger := NewGlogLogger()

	// Test that all logging levels work together
	assert.NotPanics(t, func() {
		logger.Debugf("debug")
		logger.Infof("info")
		logger.Warnf("warn")
		logger.Errorf("error")
	}, "All logging levels should work without panic")
}

func TestGlogLogger_Depth(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	glogLogger := &GlogLogger{depth: 2}

	// Test with custom depth
	assert.NotPanics(t, func() {
		glogLogger.Infof("info with custom depth")
	}, "Logger with custom depth should not panic")
}

func TestGlogLogger_EmptyMessage(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	// Test with empty messages
	assert.NotPanics(t, func() {
		logger.Infof("")
		logger.Debugf("")
		logger.Warnf("")
		logger.Errorf("")
	}, "Empty messages should not panic")
}

func TestGlogLogger_NoArgs(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	// Test logging without variadic args
	assert.NotPanics(t, func() {
		logger.Infof("simple message")
		logger.Debugf("simple debug")
		logger.Warnf("simple warning")
		logger.Errorf("simple error")
	}, "Messages without args should not panic")
}

func TestGlogLogger_MultipleArgs(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	// Test with multiple arguments
	assert.NotPanics(t, func() {
		logger.Infof("message: %s, number: %d, float: %f, bool: %v", "test", 42, 3.14, true)
	}, "Messages with multiple args should not panic")
}

func TestGlogLogger_SpecialCharacters(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	// Test with special characters
	assert.NotPanics(t, func() {
		logger.Infof("message with special chars: \n\t\"quotes\" and 'apostrophes'")
	}, "Messages with special characters should not panic")
}

// Tests for StructuredLogger interface implementation on GlogLogger

func TestGlogLogger_SlogDebug(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	logger := NewGlogLogger()

	assert.NotPanics(t, func() {
		logger.Debug("debug message")
	}, "Debug should not panic")

	assert.NotPanics(t, func() {
		logger.Debug("debug with args", "key", "value", "number", 42)
	}, "Debug with args should not panic")
}

func TestGlogLogger_SlogDebugContext(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	logger := NewGlogLogger()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.DebugContext(ctx, "debug with context")
	}, "DebugContext should not panic")

	assert.NotPanics(t, func() {
		logger.DebugContext(ctx, "debug context with args", "key", "value")
	}, "DebugContext with args should not panic")
}

func TestGlogLogger_SlogInfo(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	assert.NotPanics(t, func() {
		logger.Info("info message")
	}, "Info should not panic")

	assert.NotPanics(t, func() {
		logger.Info("info with args", "status", "ok")
	}, "Info with args should not panic")
}

func TestGlogLogger_SlogInfoContext(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()
	ctx := context.WithValue(context.Background(), "requestID", "12345")

	assert.NotPanics(t, func() {
		logger.InfoContext(ctx, "info with context")
	}, "InfoContext should not panic")

	assert.NotPanics(t, func() {
		logger.InfoContext(ctx, "info with context and args", "component", "test")
	}, "InfoContext with args should not panic")
}

func TestGlogLogger_SlogWarn(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	assert.NotPanics(t, func() {
		logger.Warn("warning message")
	}, "Warn should not panic")

	assert.NotPanics(t, func() {
		logger.Warn("warning with args", "severity", "medium")
	}, "Warn with args should not panic")
}

func TestGlogLogger_SlogWarnContext(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.WarnContext(ctx, "warning with context")
	}, "WarnContext should not panic")

	assert.NotPanics(t, func() {
		logger.WarnContext(ctx, "warning with context", "severity", "medium")
	}, "WarnContext with args should not panic")
}

func TestGlogLogger_SlogError(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	assert.NotPanics(t, func() {
		logger.Error("error message")
	}, "Error should not panic")

	assert.NotPanics(t, func() {
		logger.Error("error with details", "code", 500, "err", "internal error")
	}, "Error with args should not panic")
}

func TestGlogLogger_SlogErrorContext(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.ErrorContext(ctx, "error with context")
	}, "ErrorContext should not panic")

	assert.NotPanics(t, func() {
		logger.ErrorContext(ctx, "error with context", "component", "api")
	}, "ErrorContext with args should not panic")
}

func TestGlogLogger_SlogAllLevels(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	logger := NewGlogLogger()
	ctx := context.Background()

	// Test that all slog logging levels work together
	assert.NotPanics(t, func() {
		logger.Debug("debug")
		logger.DebugContext(ctx, "debug context")
		logger.Info("info")
		logger.InfoContext(ctx, "info context")
		logger.Warn("warn")
		logger.WarnContext(ctx, "warn context")
		logger.Error("error")
		logger.ErrorContext(ctx, "error context")
	}, "All slog logging levels should work without panic")
}

func TestGlogLogger_SlogWithVariousContexts(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger()

	// Test with different context types
	ctxWithValue := context.WithValue(context.Background(), "requestID", "abc123")
	ctxBackground := context.Background()

	assert.NotPanics(t, func() {
		logger.InfoContext(ctxWithValue, "with value context")
		logger.InfoContext(ctxBackground, "with background context")
	}, "Different context types should work")
}

func TestGlogLogger_BothGlogAndSlogMethods(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	logger := NewGlogLogger()
	ctx := context.Background()

	// Test that both old-style (Debugf, Infof) and new-style (Debug, Info) methods work
	assert.NotPanics(t, func() {
		logger.Debugf("debug formatted")
		logger.Debug("debug structured")
		logger.DebugContext(ctx, "debug with context")

		logger.Infof("info formatted: %s", "test")
		logger.Info("info structured", "key", "value")
		logger.InfoContext(ctx, "info with context")

		logger.Warnf("warn formatted")
		logger.Warn("warn structured")
		logger.WarnContext(ctx, "warn with context")

		logger.Errorf("error formatted: %v", "error")
		logger.Error("error structured", "err", "error")
		logger.ErrorContext(ctx, "error with context")
	}, "Both GlogLogger and SlogLogger methods should work on same instance")
}

func TestGlogLogger_FatalCallsExit(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger().(*GlogLogger)

	// Track whether exit was called and with what code
	exitCalled := false
	exitCode := -1

	// Override the exit function for testing
	logger.exitFunc = func(code int) {
		exitCalled = true
		exitCode = code
	}

	// Call Fatal
	logger.Fatal("fatal error message")

	// Verify exit was called with code 1
	assert.True(t, exitCalled, "Fatal should call exit function")
	assert.Equal(t, 1, exitCode, "Fatal should exit with code 1")
}

func TestGlogLogger_FatalContextCallsExit(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger().(*GlogLogger)
	ctx := context.Background()

	// Track whether exit was called and with what code
	exitCalled := false
	exitCode := -1

	// Override the exit function for testing
	logger.exitFunc = func(code int) {
		exitCalled = true
		exitCode = code
	}

	// Call FatalContext
	logger.FatalContext(ctx, "fatal error with context", "key", "value")

	// Verify exit was called with code 1
	assert.True(t, exitCalled, "FatalContext should call exit function")
	assert.Equal(t, 1, exitCode, "FatalContext should exit with code 1")
}

func TestGlogLogger_FatalContextWithCustomContext(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	logger := NewGlogLogger().(*GlogLogger)
	ctx := context.WithValue(context.Background(), "requestID", "test-123")

	// Track whether exit was called
	exitCalled := false

	// Override the exit function for testing
	logger.exitFunc = func(code int) {
		exitCalled = true
	}

	// Call FatalContext with custom context
	logger.FatalContext(ctx, "fatal with custom context")

	// Verify exit was called
	assert.True(t, exitCalled, "FatalContext should call exit function even with custom context")
}
