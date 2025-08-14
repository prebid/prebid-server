package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSlogLogger_NewSlogLogger tests the NewSlogLogger constructor
func TestSlogLogger_NewSlogLogger(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates new slog logger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewSlogLogger()

			assert.NotNil(t, logger)
			assert.Implements(t, (*Logger)(nil), logger)

			// Type assertion to check internal state
			slogLogger, ok := logger.(*SlogLogger)
			assert.True(t, ok)
			assert.NotNil(t, slogLogger.logger)
		})
	}
}

// TestSlogLogger_Debug tests the Debug method
func TestSlogLogger_Debug(t *testing.T) {
	logger := NewSlogLogger()

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
			name: "message with key-value pairs",
			msg:  "test debug message",
			args: []interface{}{"key", "value", "number", 42},
		},
		{
			name: "empty message",
			msg:  "",
			args: nil,
		},
		{
			name: "message with args",
			msg:  "test message",
			args: []interface{}{"extra", "args"},
		},
		{
			name: "message with mixed types",
			msg:  "mixed types",
			args: []interface{}{"string", "value", "int", 123, "bool", true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test mainly ensures no panic occurs
			// Since slog writes to stdout, we can't easily capture it in tests
			// but we can verify the method executes without error
			assert.NotPanics(t, func() {
				logger.Debug(tt.msg, tt.args...)
			})
		})
	}
}

// TestSlogLogger_DebugContext tests the DebugContext method
func TestSlogLogger_DebugContext(t *testing.T) {
	logger := NewSlogLogger()

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
			msg:  "test debug message",
			args: []interface{}{"key", "context", "number", 123},
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

// TestSlogLogger_Info tests the Info method
func TestSlogLogger_Info(t *testing.T) {
	logger := NewSlogLogger()

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
			name: "info message with key-value pairs",
			msg:  "info message",
			args: []interface{}{"event", "happened", "timestamp", "2023-01-01"},
		},
		{
			name: "info with multiple args",
			msg:  "processing items",
			args: []interface{}{"count", 10, "type", "urgent", "priority", 9.5},
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

// TestSlogLogger_InfoContext tests the InfoContext method
func TestSlogLogger_InfoContext(t *testing.T) {
	logger := NewSlogLogger()

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

// TestSlogLogger_Warn tests the Warn method
func TestSlogLogger_Warn(t *testing.T) {
	logger := NewSlogLogger()

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
			msg:  "operation failed",
			args: []interface{}{"operation", "save", "code", 500},
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

// TestSlogLogger_WarnContext tests the WarnContext method
func TestSlogLogger_WarnContext(t *testing.T) {
	logger := NewSlogLogger()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.WarnContext(ctx, "contextual warning", "details", "value")
	})

	assert.NotPanics(t, func() {
		logger.WarnContext(nil, "warning with nil context")
	})
}

// TestSlogLogger_Error tests the Error method
func TestSlogLogger_Error(t *testing.T) {
	logger := NewSlogLogger()

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
			msg:  "failed to connect",
			args: []interface{}{"reason", "network unavailable"},
		},
		{
			name: "error with numeric codes",
			msg:  "not found",
			args: []interface{}{"code", 404, "resource", "user"},
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

// TestSlogLogger_ErrorContext tests the ErrorContext method
func TestSlogLogger_ErrorContext(t *testing.T) {
	logger := NewSlogLogger()
	ctx := context.Background()

	assert.NotPanics(t, func() {
		logger.ErrorContext(ctx, "contextual error", "reason", "database connection failed")
	})

	assert.NotPanics(t, func() {
		logger.ErrorContext(nil, "error with nil context")
	})
}

// TestSlogLogger_InterfaceCompliance tests that SlogLogger implements the Logger interface
func TestSlogLogger_InterfaceCompliance(t *testing.T) {
	var logger Logger = NewSlogLogger()
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

// TestSlogLogger_ConcurrentUsage tests concurrent usage of the logger
func TestSlogLogger_ConcurrentUsage(t *testing.T) {
	logger := NewSlogLogger()
	ctx := context.Background()

	// Run multiple goroutines concurrently
	const numGoroutines = 10
	const messagesPerGoroutine = 5

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Debug("goroutine message", "goroutine", id, "message", j)
				logger.Info("goroutine info", "goroutine", id, "message", j)
				logger.WarnContext(ctx, "goroutine warn", "goroutine", id, "message", j)
				logger.ErrorContext(ctx, "goroutine error", "goroutine", id, "message", j)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestSlogLogger_NilArgs tests behavior with nil arguments
func TestSlogLogger_NilArgs(t *testing.T) {
	logger := NewSlogLogger()

	assert.NotPanics(t, func() {
		logger.Debug("test with nil args", nil)
		logger.Info("test with nil args", nil)
		logger.Warn("test with nil args", nil)
		logger.Error("test with nil args", nil)
	})
}

// TestSlogLogger_EmptyArgs tests behavior with empty argument slice
func TestSlogLogger_EmptyArgs(t *testing.T) {
	logger := NewSlogLogger()

	assert.NotPanics(t, func() {
		logger.Debug("test with empty args")
		logger.Info("test with empty args")
		logger.Warn("test with empty args")
		logger.Error("test with empty args")
	})
}

// TestSlogLogger_OddNumberOfArgs tests behavior with odd number of arguments
func TestSlogLogger_OddNumberOfArgs(t *testing.T) {
	logger := NewSlogLogger()

	assert.NotPanics(t, func() {
		logger.Debug("test with odd args", "key1", "value1", "key2")
		logger.Info("test with odd args", "key1", "value1", "key2")
		logger.Warn("test with odd args", "key1", "value1", "key2")
		logger.Error("test with odd args", "key1", "value1", "key2")
	})
}

// TestSlogLogger_ContextValues tests that context values are properly handled
func TestSlogLogger_ContextValues(t *testing.T) {
	logger := NewSlogLogger()

	// Test with nil context
	assert.NotPanics(t, func() {
		logger.DebugContext(nil, "test nil context")
		logger.InfoContext(nil, "test nil context")
		logger.WarnContext(nil, "test nil context")
		logger.ErrorContext(nil, "test nil context")
	})

	// Test with context containing values
	ctx := context.WithValue(context.Background(), "requestID", "12345")
	assert.NotPanics(t, func() {
		logger.DebugContext(ctx, "test with context")
		logger.InfoContext(ctx, "test with context")
		logger.WarnContext(ctx, "test with context")
		logger.ErrorContext(ctx, "test with context")
	})
}

// BenchmarkSlogLogger_Debug benchmarks the Debug method
func BenchmarkSlogLogger_Debug(b *testing.B) {
	logger := NewSlogLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("benchmark debug message", "iteration", i)
	}
}

// BenchmarkSlogLogger_Info benchmarks the Info method
func BenchmarkSlogLogger_Info(b *testing.B) {
	logger := NewSlogLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark info message", "iteration", i)
	}
}

// BenchmarkSlogLogger_Error benchmarks the Error method
func BenchmarkSlogLogger_Error(b *testing.B) {
	logger := NewSlogLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("benchmark error message", "iteration", i)
	}
}

// BenchmarkSlogLogger_InfoContext benchmarks the InfoContext method
func BenchmarkSlogLogger_InfoContext(b *testing.B) {
	logger := NewSlogLogger()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "benchmark info context message", "iteration", i)
	}
}
