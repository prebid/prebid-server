package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockLogger is a test implementation of the Logger interface
type mockLogger struct {
	debugCalls    []logCall
	infoCalls     []logCall
	warnCalls     []logCall
	errorCalls    []logCall
	debugCtxCalls []logCtxCall
	infoCtxCalls  []logCtxCall
	warnCtxCalls  []logCtxCall
	errorCtxCalls []logCtxCall
}

type logCall struct {
	msg  any
	args []any
}

type logCtxCall struct {
	ctx  context.Context
	msg  any
	args []any
}

func (m *mockLogger) Debug(msg any, args ...any) {
	m.debugCalls = append(m.debugCalls, logCall{msg: msg, args: args})
}

func (m *mockLogger) DebugContext(ctx context.Context, msg any, args ...any) {
	m.debugCtxCalls = append(m.debugCtxCalls, logCtxCall{ctx: ctx, msg: msg, args: args})
}

func (m *mockLogger) Info(msg any, args ...any) {
	m.infoCalls = append(m.infoCalls, logCall{msg: msg, args: args})
}

func (m *mockLogger) InfoContext(ctx context.Context, msg any, args ...any) {
	m.infoCtxCalls = append(m.infoCtxCalls, logCtxCall{ctx: ctx, msg: msg, args: args})
}

func (m *mockLogger) Warn(msg any, args ...any) {
	m.warnCalls = append(m.warnCalls, logCall{msg: msg, args: args})
}

func (m *mockLogger) WarnContext(ctx context.Context, msg any, args ...any) {
	m.warnCtxCalls = append(m.warnCtxCalls, logCtxCall{ctx: ctx, msg: msg, args: args})
}

func (m *mockLogger) Error(msg any, args ...any) {
	m.errorCalls = append(m.errorCalls, logCall{msg: msg, args: args})
}

func (m *mockLogger) ErrorContext(ctx context.Context, msg any, args ...any) {
	m.errorCtxCalls = append(m.errorCtxCalls, logCtxCall{ctx: ctx, msg: msg, args: args})
}

func setupTest() {
	// Reset global logger to default before each test
	logger = NewSlogLogger()
}

// TestNew tests the New function with various configurations
func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		loggerType  string
		depth       *int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "slog logger",
			loggerType:  "slog",
			depth:       nil,
			expectError: false,
		},
		{
			name:        "slog logger with depth",
			loggerType:  "slog",
			depth:       intPtr(2),
			expectError: false,
		},
		{
			name:        "glog logger",
			loggerType:  "glog",
			depth:       nil,
			expectError: false,
		},
		{
			name:        "glog logger with depth",
			loggerType:  "glog",
			depth:       intPtr(5),
			expectError: false,
		},
		{
			name:        "custom logger without instance",
			loggerType:  "custom",
			depth:       nil,
			expectError: true,
			errorMsg:    "custom logger type requires CustomLogger instance",
		},
		{
			name:        "unsupported logger type",
			loggerType:  "invalid",
			depth:       nil,
			expectError: true,
			errorMsg:    "unsupported logger type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()

			err := New(tt.loggerType, tt.depth)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, GetCurrentLogger())
			}
		})
	}
}

// TestNewWithConfig tests the NewWithConfig function
func TestNewWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *LoggerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "logger config is nil",
		},
		{
			name: "slog config",
			config: &LoggerConfig{
				Type: LoggerTypeSlog,
			},
			expectError: false,
		},
		{
			name: "glog config",
			config: &LoggerConfig{
				Type:  LoggerTypeGlog,
				Depth: intPtr(3),
			},
			expectError: false,
		},
		{
			name: "custom config with logger",
			config: &LoggerConfig{
				Type:         LoggerTypeCustom,
				CustomLogger: &mockLogger{},
			},
			expectError: false,
		},
		{
			name: "custom config without logger",
			config: &LoggerConfig{
				Type: LoggerTypeCustom,
			},
			expectError: true,
			errorMsg:    "custom logger type requires CustomLogger instance",
		},
		{
			name: "unsupported type",
			config: &LoggerConfig{
				Type: LoggerType("unknown"),
			},
			expectError: true,
			errorMsg:    "unsupported logger type: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()

			err := NewWithConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, GetCurrentLogger())
			}
		})
	}
}

// TestSetCustomLogger tests the SetCustomLogger function
func TestSetCustomLogger(t *testing.T) {
	setupTest()

	customLogger := &mockLogger{}
	SetCustomLogger(customLogger)

	currentLogger := GetCurrentLogger()
	assert.Equal(t, customLogger, currentLogger)
}

// TestGetCurrentLogger tests the GetCurrentLogger function
func TestGetCurrentLogger(t *testing.T) {
	setupTest()

	currentLogger := GetCurrentLogger()
	assert.NotNil(t, currentLogger)
	assert.Implements(t, (*Logger)(nil), currentLogger)
}

// TestGlobalDebugFunctions tests the global Debug functions
func TestGlobalDebugFunctions(t *testing.T) {
	mock := &mockLogger{}
	SetCustomLogger(mock)

	tests := []struct {
		name string
		fn   func()
		ctx  context.Context
	}{
		{
			name: "Debug",
			fn: func() {
				Debug("test debug", "key", "value")
			},
		},
		{
			name: "DebugContext",
			fn: func() {
				DebugContext(context.Background(), "test debug context", "key", "value")
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.debugCalls = nil
			mock.debugCtxCalls = nil

			tt.fn()

			if tt.ctx != nil {
				assert.Len(t, mock.debugCtxCalls, 1)
				assert.Equal(t, "test debug context", mock.debugCtxCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.debugCtxCalls[0].args)
			} else {
				assert.Len(t, mock.debugCalls, 1)
				assert.Equal(t, "test debug", mock.debugCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.debugCalls[0].args)
			}
		})
	}
}

// TestGlobalInfoFunctions tests the global Info functions
func TestGlobalInfoFunctions(t *testing.T) {
	mock := &mockLogger{}
	SetCustomLogger(mock)

	tests := []struct {
		name string
		fn   func()
		ctx  context.Context
	}{
		{
			name: "Info",
			fn: func() {
				Info("test info", "key", "value")
			},
		},
		{
			name: "InfoContext",
			fn: func() {
				InfoContext(context.Background(), "test info context", "key", "value")
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.infoCalls = nil
			mock.infoCtxCalls = nil

			tt.fn()

			if tt.ctx != nil {
				assert.Len(t, mock.infoCtxCalls, 1)
				assert.Equal(t, "test info context", mock.infoCtxCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.infoCtxCalls[0].args)
			} else {
				assert.Len(t, mock.infoCalls, 1)
				assert.Equal(t, "test info", mock.infoCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.infoCalls[0].args)
			}
		})
	}
}

// TestGlobalWarnFunctions tests the global Warn functions
func TestGlobalWarnFunctions(t *testing.T) {
	mock := &mockLogger{}
	SetCustomLogger(mock)

	tests := []struct {
		name string
		fn   func()
		ctx  context.Context
	}{
		{
			name: "Warn",
			fn: func() {
				Warn("test warn", "key", "value")
			},
		},
		{
			name: "WarnContext",
			fn: func() {
				WarnContext(context.Background(), "test warn context", "key", "value")
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.warnCalls = nil
			mock.warnCtxCalls = nil

			tt.fn()

			if tt.ctx != nil {
				assert.Len(t, mock.warnCtxCalls, 1)
				assert.Equal(t, "test warn context", mock.warnCtxCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.warnCtxCalls[0].args)
			} else {
				assert.Len(t, mock.warnCalls, 1)
				assert.Equal(t, "test warn", mock.warnCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.warnCalls[0].args)
			}
		})
	}
}

// TestGlobalErrorFunctions tests the global Error functions
func TestGlobalErrorFunctions(t *testing.T) {
	mock := &mockLogger{}
	SetCustomLogger(mock)

	tests := []struct {
		name string
		fn   func()
		ctx  context.Context
	}{
		{
			name: "Error",
			fn: func() {
				Error("test error", "key", "value")
			},
		},
		{
			name: "ErrorContext",
			fn: func() {
				ErrorContext(context.Background(), "test error context", "key", "value")
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.errorCalls = nil
			mock.errorCtxCalls = nil

			tt.fn()

			if tt.ctx != nil {
				assert.Len(t, mock.errorCtxCalls, 1)
				assert.Equal(t, "test error context", mock.errorCtxCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.errorCtxCalls[0].args)
			} else {
				assert.Len(t, mock.errorCalls, 1)
				assert.Equal(t, "test error", mock.errorCalls[0].msg)
				assert.Equal(t, []any{"key", "value"}, mock.errorCalls[0].args)
			}
		})
	}
}

// TestLoggerTypes tests the LoggerType constants
func TestLoggerTypes(t *testing.T) {
	assert.Equal(t, LoggerType("slog"), LoggerTypeSlog)
	assert.Equal(t, LoggerType("glog"), LoggerTypeGlog)
	assert.Equal(t, LoggerType("custom"), LoggerTypeCustom)
}

// TestDefaultDepth tests the default depth value
func TestDefaultDepth(t *testing.T) {
	assert.Equal(t, 1, defaultDepth)
}

// TestLoggerConfigDepthHandling tests depth handling in LoggerConfig
func TestLoggerConfigDepthHandling(t *testing.T) {
	tests := []struct {
		name          string
		config        *LoggerConfig
		expectedDepth int
	}{
		{
			name: "nil depth uses default",
			config: &LoggerConfig{
				Type: LoggerTypeGlog,
			},
			expectedDepth: defaultDepth,
		},
		{
			name: "provided depth is used",
			config: &LoggerConfig{
				Type:  LoggerTypeGlog,
				Depth: intPtr(5),
			},
			expectedDepth: 5,
		},
		{
			name: "zero depth is preserved",
			config: &LoggerConfig{
				Type:  LoggerTypeGlog,
				Depth: intPtr(0),
			},
			expectedDepth: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()

			err := NewWithConfig(tt.config)
			assert.NoError(t, err)

			// For glog logger, we can verify the depth was set correctly
			if tt.config.Type == LoggerTypeGlog {
				glogLogger, ok := GetCurrentLogger().(*GlogLogger)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedDepth, glogLogger.depth)
			}
		})
	}
}

// TestConcurrentAccess tests concurrent access to global logger functions
func TestConcurrentAccess(t *testing.T) {
	setupTest()

	const numGoroutines = 10
	const messagesPerGoroutine = 5

	done := make(chan bool, numGoroutines)
	ctx := context.Background()

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < messagesPerGoroutine; j++ {
				Debug("concurrent debug", "goroutine", id, "message", j)
				Info("concurrent info", "goroutine", id, "message", j)
				Warn("concurrent warn", "goroutine", id, "message", j)
				Error("concurrent error", "goroutine", id, "message", j)

				DebugContext(ctx, "concurrent debug context", "goroutine", id, "message", j)
				InfoContext(ctx, "concurrent info context", "goroutine", id, "message", j)
				WarnContext(ctx, "concurrent warn context", "goroutine", id, "message", j)
				ErrorContext(ctx, "concurrent error context", "goroutine", id, "message", j)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// If we get here without panic, concurrent access is working
	assert.True(t, true)
}

// TestLoggerSwitching tests switching between different logger types
func TestLoggerSwitching(t *testing.T) {
	// Start with slog
	err := NewWithConfig(&LoggerConfig{Type: LoggerTypeSlog})
	assert.NoError(t, err)
	_, ok := GetCurrentLogger().(*SlogLogger)
	assert.True(t, ok)

	// Switch to glog
	err = NewWithConfig(&LoggerConfig{Type: LoggerTypeGlog, Depth: intPtr(2)})
	assert.NoError(t, err)
	glogLogger, ok := GetCurrentLogger().(*GlogLogger)
	assert.True(t, ok)
	assert.Equal(t, 2, glogLogger.depth)

	// Switch to custom
	customLogger := &mockLogger{}
	err = NewWithConfig(&LoggerConfig{Type: LoggerTypeCustom, CustomLogger: customLogger})
	assert.NoError(t, err)
	assert.Equal(t, customLogger, GetCurrentLogger())
}

// TestGlobalFunctionsWithVariousArgs tests global functions with various argument types
func TestGlobalFunctionsWithVariousArgs(t *testing.T) {
	mock := &mockLogger{}
	SetCustomLogger(mock)

	// Test with no args
	Info("no args message")
	assert.Len(t, mock.infoCalls, 1)
	assert.Equal(t, "no args message", mock.infoCalls[0].msg)
	assert.Nil(t, mock.infoCalls[0].args)

	// Reset and test with multiple args
	mock.infoCalls = nil
	Info("with args", "key1", "value1", "key2", 42, "key3", true)
	assert.Len(t, mock.infoCalls, 1)
	assert.Equal(t, "with args", mock.infoCalls[0].msg)
	assert.Equal(t, []any{"key1", "value1", "key2", 42, "key3", true}, mock.infoCalls[0].args)
}

// TestContextPassing tests that context is properly passed to context methods
func TestContextPassing(t *testing.T) {
	mock := &mockLogger{}
	SetCustomLogger(mock)

	ctx := context.WithValue(context.Background(), "testKey", "testValue")

	InfoContext(ctx, "test message", "arg", "value")

	assert.Len(t, mock.infoCtxCalls, 1)
	assert.Equal(t, ctx, mock.infoCtxCalls[0].ctx)
	assert.Equal(t, "test message", mock.infoCtxCalls[0].msg)
	assert.Equal(t, []any{"arg", "value"}, mock.infoCtxCalls[0].args)
}

// TestNilContextHandling tests behavior with nil context
func TestNilContextHandling(t *testing.T) {
	mock := &mockLogger{}
	SetCustomLogger(mock)

	InfoContext(nil, "test message with nil context")

	assert.Len(t, mock.infoCtxCalls, 1)
	assert.Nil(t, mock.infoCtxCalls[0].ctx)
	assert.Equal(t, "test message with nil context", mock.infoCtxCalls[0].msg)
}

// BenchmarkGlobalInfo benchmarks the global Info function
func BenchmarkGlobalInfo(b *testing.B) {
	setupTest()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", "iteration", i)
	}
}

// BenchmarkGlobalInfoContext benchmarks the global InfoContext function
func BenchmarkGlobalInfoContext(b *testing.B) {
	setupTest()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InfoContext(ctx, "benchmark message", "iteration", i)
	}
}

// BenchmarkLoggerSwitching benchmarks switching logger types
func BenchmarkLoggerSwitching(b *testing.B) {
	configs := []*LoggerConfig{
		{Type: LoggerTypeSlog},
		{Type: LoggerTypeGlog, Depth: intPtr(1)},
		{Type: LoggerTypeCustom, CustomLogger: &mockLogger{}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := configs[i%len(configs)]
		NewWithConfig(config)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
