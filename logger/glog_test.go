package logger

import (
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
