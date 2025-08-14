package logger

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGlogLogger_NewGlogLogger tests the NewGlogLogger constructor
func TestGlogLogger_NewGlogLogger(t *testing.T) {
	tests := []struct {
		name  string
		depth int
	}{
		{
			name:  "positive depth",
			depth: 5,
		},
		{
			name:  "zero depth",
			depth: 0,
		},
		{
			name:  "negative depth",
			depth: -1,
		},
		{
			name:  "large depth",
			depth: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewGlogLogger(tt.depth)

			assert.NotNil(t, logger)
			assert.Implements(t, (*Logger)(nil), logger)

			// Type assertion to check internal state
			glogLogger, ok := logger.(*GlogLogger)
			assert.True(t, ok)
			assert.Equal(t, tt.depth, glogLogger.depth)
		})
	}
}

// TestGlogLogger_Debug tests the Debug method
func TestGlogLogger_Debug(t *testing.T) {
	logger := NewGlogLogger(1)

	tests := []struct {
		name string
		msg  string
		args []interface{}
	}{
		{
			name: "simple message",
			msg:  "test debug message",
			args: nil,
		},
		{
			name: "message with format args",
			msg:  "test debug message with %s and %d",
			args: []interface{}{"string", 42},
		},
		{
			name: "empty message",
			msg:  "",
			args: nil,
		},
		{
			name: "message with no format but args provided",
			msg:  "test message",
			args: []interface{}{"extra", "args"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test mainly ensures no panic occurs
			// Since glog is a third-party library, we can't easily mock it
			// but we can verify the method executes without error
			assert.NotPanics(t, func() {
				logger.Debug(tt.msg, tt.args...)
			})
		})
	}
}

// TestGlogLogger_DebugContext tests the DebugContext method
func TestGlogLogger_DebugContext(t *testing.T) {
	logger := NewGlogLogger(1)

	tests := []struct {
		name string
		ctx  context.Context
		msg  string
		args []interface{}
	}{
		{
			name: "with background context",
			ctx:  context.Background(),
			msg:  "test debug message with context",
			args: nil,
		},
		{
			name: "with nil context",
			ctx:  nil,
			msg:  "test debug message with nil context",
			args: nil,
		},
		{
			name: "with context and args",
			ctx:  context.Background(),
			msg:  "test debug message with %s and %d",
			args: []interface{}{"context", 123},
		},
		{
			name: "with context value",
			ctx:  context.WithValue(context.Background(), "key", "value"),
			msg:  "test debug message with context value",
			args: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				logger.DebugContext(tt.ctx, tt.msg, tt.args...)
			})
		})
	}
}

// TestGlogLogger_Info tests the Info method
func TestGlogLogger_Info(t *testing.T) {
	logger := NewGlogLogger(2)

	tests := []struct {
		name string
		msg  string
		args []interface{}
	}{
		{
			name: "simple info message",
			msg:  "test info message",
			args: nil,
		},
		{
			name: "info message with format",
			msg:  "info: %s happened at %v",
			args: []interface{}{"event", "2023-01-01"},
		},
		{
			name: "info with multiple args",
			msg:  "processing %d items of type %s with priority %f",
			args: []interface{}{10, "urgent", 9.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				logger.Info(tt.msg, tt.args...)
			})
		})
	}
}

// TestGlogLogger_InfoContext tests the InfoContext method
func TestGlogLogger_InfoContext(t *testing.T) {
	logger := NewGlogLogger(2)

	tests := []struct {
		name string
		ctx  context.Context
		msg  string
		args []interface{}
	}{
		{
			name: "info with context",
			ctx:  context.Background(),
			msg:  "contextual info message",
			args: nil,
		},
		{
			name: "info with timeout context",
			ctx:  func() context.Context { ctx, _ := context.WithCancel(context.Background()); return ctx }(),
			msg:  "info with timeout context",
			args: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				logger.InfoContext(tt.ctx, tt.msg, tt.args...)
			})
		})
	}
}

// TestGlogLogger_Warn tests the Warn method
func TestGlogLogger_Warn(t *testing.T) {
	logger := NewGlogLogger(3)

	tests := []struct {
		name string
		msg  string
		args []interface{}
	}{
		{
			name: "simple warning",
			msg:  "this is a warning",
			args: nil,
		},
		{
			name: "warning with details",
			msg:  "warning: operation %s failed with code %d",
			args: []interface{}{"save", 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				logger.Warn(tt.msg, tt.args...)
			})
		})
	}
}

// TestGlogLogger_WarnContext tests the WarnContext method
func TestGlogLogger_WarnContext(t *testing.T) {
	logger := NewGlogLogger(3)
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.WarnContext(ctx, "contextual warning with %s", "details")
	})

	assert.NotPanics(t, func() {
		logger.WarnContext(nil, "warning with nil context")
	})
}

// TestGlogLogger_Error tests the Error method
func TestGlogLogger_Error(t *testing.T) {
	logger := NewGlogLogger(4)

	tests := []struct {
		name string
		msg  string
		args []interface{}
	}{
		{
			name: "simple error",
			msg:  "an error occurred",
			args: nil,
		},
		{
			name: "error with details",
			msg:  "error: failed to %s because %s",
			args: []interface{}{"connect", "network unavailable"},
		},
		{
			name: "error with numeric codes",
			msg:  "error code %d: %s",
			args: []interface{}{404, "not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				logger.Error(tt.msg, tt.args...)
			})
		})
	}
}

// TestGlogLogger_ErrorContext tests the ErrorContext method
func TestGlogLogger_ErrorContext(t *testing.T) {
	logger := NewGlogLogger(4)
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.ErrorContext(ctx, "contextual error: %s", "database connection failed")
	})

	assert.NotPanics(t, func() {
		logger.ErrorContext(nil, "error with nil context")
	})
}

// TestGlogLogger_InterfaceCompliance tests that GlogLogger implements the Logger interface
func TestGlogLogger_InterfaceCompliance(t *testing.T) {
	var logger Logger = NewGlogLogger(1)
	assert.NotNil(t, logger)

	// Test that all interface methods are callable
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.Debug("debug", "test")
		logger.DebugContext(ctx, "debug context", "test")
		logger.Info("info", "test")
		logger.InfoContext(ctx, "info context", "test")
		logger.Warn("warn", "test")
		logger.WarnContext(ctx, "warn context", "test")
		logger.Error("error", "test")
		logger.ErrorContext(ctx, "error context", "test")
	})
}

// TestGlogLogger_DifferentDepths tests logger behavior with different depth values
func TestGlogLogger_DifferentDepths(t *testing.T) {
	depths := []int{0, 1, 2, 5, 10}

	for _, depth := range depths {
		t.Run(fmt.Sprintf("depth_%d", depth), func(t *testing.T) {
			logger := NewGlogLogger(depth)
			glogLogger := logger.(*GlogLogger)

			assert.Equal(t, depth, glogLogger.depth)

			// Test that all methods work regardless of depth
			assert.NotPanics(t, func() {
				logger.Debug("test debug at depth %d", depth)
				logger.Info("test info at depth %d", depth)
				logger.Warn("test warn at depth %d", depth)
				logger.Error("test error at depth %d", depth)
			})
		})
	}
}

// TestGlogLogger_ConcurrentUsage tests concurrent usage of the logger
func TestGlogLogger_ConcurrentUsage(t *testing.T) {
	logger := NewGlogLogger(1)
	ctx := context.Background()

	// Run multiple goroutines concurrently
	const numGoroutines = 10
	const messagesPerGoroutine = 5

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Debug("goroutine %d message %d", id, j)
				logger.Info("goroutine %d info %d", id, j)
				logger.WarnContext(ctx, "goroutine %d warn %d", id, j)
				logger.ErrorContext(ctx, "goroutine %d error %d", id, j)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestGlogLogger_NilArgs tests behavior with nil arguments
func TestGlogLogger_NilArgs(t *testing.T) {
	logger := NewGlogLogger(1)

	assert.NotPanics(t, func() {
		logger.Debug("test with nil args", nil)
		logger.Info("test with nil args", nil)
		logger.Warn("test with nil args", nil)
		logger.Error("test with nil args", nil)
	})
}

// TestGlogLogger_EmptyArgs tests behavior with empty argument slice
func TestGlogLogger_EmptyArgs(t *testing.T) {
	logger := NewGlogLogger(1)

	assert.NotPanics(t, func() {
		logger.Debug("test with empty args")
		logger.Info("test with empty args")
		logger.Warn("test with empty args")
		logger.Error("test with empty args")
	})
}

// BenchmarkGlogLogger_Debug benchmarks the Debug method
func BenchmarkGlogLogger_Debug(b *testing.B) {
	logger := NewGlogLogger(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("benchmark debug message %d", i)
	}
}

// BenchmarkGlogLogger_Info benchmarks the Info method
func BenchmarkGlogLogger_Info(b *testing.B) {
	logger := NewGlogLogger(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark info message %d", i)
	}
}

// BenchmarkGlogLogger_Error benchmarks the Error method
func BenchmarkGlogLogger_Error(b *testing.B) {
	logger := NewGlogLogger(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("benchmark error message %d", i)
	}
}
