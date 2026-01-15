package logger

import (
	"context"
	"flag"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockLogger is a test implementation of the Logger interface
type mockLogger struct {
	debugCalls        []logCall
	infoCalls         []logCall
	warnCalls         []logCall
	errorCalls        []logCall
	fatalCalls        []logCall
	debugContextCalls []contextLogCall
	infoContextCalls  []contextLogCall
	warnContextCalls  []contextLogCall
	errorContextCalls []contextLogCall
	fatalContextCalls []contextLogCall
}

type logCall struct {
	msg  string
	args []any
}

type contextLogCall struct {
	ctx  context.Context
	msg  string
	args []any
}

func (m *mockLogger) Debugf(msg string, args ...any) {
	m.debugCalls = append(m.debugCalls, logCall{msg, args})
}

func (m *mockLogger) Infof(msg string, args ...any) {
	m.infoCalls = append(m.infoCalls, logCall{msg, args})
}

func (m *mockLogger) Warnf(msg string, args ...any) {
	m.warnCalls = append(m.warnCalls, logCall{msg, args})
}

func (m *mockLogger) Errorf(msg string, args ...any) {
	m.errorCalls = append(m.errorCalls, logCall{msg, args})
}

func (m *mockLogger) Fatalf(msg string, args ...any) {
	m.fatalCalls = append(m.fatalCalls, logCall{msg, args})
}

// StructuredLogger interface implementation for mockLogger

func (m *mockLogger) Debug(msg string, args ...any) {
	m.debugCalls = append(m.debugCalls, logCall{msg, args})
}

func (m *mockLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	m.debugContextCalls = append(m.debugContextCalls, contextLogCall{ctx, msg, args})
}

func (m *mockLogger) Info(msg string, args ...any) {
	m.infoCalls = append(m.infoCalls, logCall{msg, args})
}

func (m *mockLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	m.infoContextCalls = append(m.infoContextCalls, contextLogCall{ctx, msg, args})
}

func (m *mockLogger) Warn(msg string, args ...any) {
	m.warnCalls = append(m.warnCalls, logCall{msg, args})
}

func (m *mockLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	m.warnContextCalls = append(m.warnContextCalls, contextLogCall{ctx, msg, args})
}

func (m *mockLogger) Error(msg string, args ...any) {
	m.errorCalls = append(m.errorCalls, logCall{msg, args})
}

func (m *mockLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	m.errorContextCalls = append(m.errorContextCalls, contextLogCall{ctx, msg, args})
}

func (m *mockLogger) Fatal(msg string, args ...any) {
	m.fatalCalls = append(m.fatalCalls, logCall{msg, args})
}

func (m *mockLogger) FatalContext(ctx context.Context, msg string, args ...any) {
	m.fatalContextCalls = append(m.fatalContextCalls, contextLogCall{ctx, msg, args})
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		debugCalls:        []logCall{},
		infoCalls:         []logCall{},
		warnCalls:         []logCall{},
		errorCalls:        []logCall{},
		fatalCalls:        []logCall{},
		debugContextCalls: []contextLogCall{},
		infoContextCalls:  []contextLogCall{},
		warnContextCalls:  []contextLogCall{},
		errorContextCalls: []contextLogCall{},
		fatalContextCalls: []contextLogCall{},
	}
}

func TestDefaultLogger(t *testing.T) {
	// The default logger should be GlogLogger
	defaultLogger := logger
	assert.NotNil(t, defaultLogger, "Default logger should not be nil")

	_, ok := defaultLogger.(*GlogLogger)
	assert.True(t, ok, "Default logger should be *GlogLogger")
}

func TestDebug(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	mock := newMockLogger()
	logger = mock

	Debugf("debug message")
	assert.Len(t, mock.debugCalls, 1, "Should have one debug call")
	assert.Equal(t, "debug message", mock.debugCalls[0].msg)
	assert.Empty(t, mock.debugCalls[0].args)

	Debugf("debug with args: %s, %d", "test", 123)
	assert.Len(t, mock.debugCalls, 2, "Should have two debug calls")
	assert.Equal(t, "debug with args: %s, %d", mock.debugCalls[1].msg)
	assert.Equal(t, []any{"test", 123}, mock.debugCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestInfo(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Infof("info message")
	assert.Len(t, mock.infoCalls, 1, "Should have one info call")
	assert.Equal(t, "info message", mock.infoCalls[0].msg)
	assert.Empty(t, mock.infoCalls[0].args)

	Infof("info with args: %s, %d", "test", 456)
	assert.Len(t, mock.infoCalls, 2, "Should have two info calls")
	assert.Equal(t, "info with args: %s, %d", mock.infoCalls[1].msg)
	assert.Equal(t, []any{"test", 456}, mock.infoCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestWarn(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Warnf("warning message")
	assert.Len(t, mock.warnCalls, 1, "Should have one warn call")
	assert.Equal(t, "warning message", mock.warnCalls[0].msg)
	assert.Empty(t, mock.warnCalls[0].args)

	Warnf("warning with args: %s, %d", "test", 789)
	assert.Len(t, mock.warnCalls, 2, "Should have two warn calls")
	assert.Equal(t, "warning with args: %s, %d", mock.warnCalls[1].msg)
	assert.Equal(t, []any{"test", 789}, mock.warnCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestError(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Errorf("error message")
	assert.Len(t, mock.errorCalls, 1, "Should have one error call")
	assert.Equal(t, "error message", mock.errorCalls[0].msg)
	assert.Empty(t, mock.errorCalls[0].args)

	Errorf("error with args: %s, %d", "test", 999)
	assert.Len(t, mock.errorCalls, 2, "Should have two error calls")
	assert.Equal(t, "error with args: %s, %d", mock.errorCalls[1].msg)
	assert.Equal(t, []any{"test", 999}, mock.errorCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestAllLogLevels(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	mock := newMockLogger()
	logger = mock

	Debugf("debug")
	Infof("info")
	Warnf("warn")
	Errorf("error")
	Fatalf("fatal")

	assert.Len(t, mock.debugCalls, 1, "Should have one debug call")
	assert.Len(t, mock.infoCalls, 1, "Should have one info call")
	assert.Len(t, mock.warnCalls, 1, "Should have one warn call")
	assert.Len(t, mock.errorCalls, 1, "Should have one error call")
	assert.Len(t, mock.fatalCalls, 1, "Should have one fatal call")

	// Restore default logger
	logger = NewGlogLogger()
}

func TestEmptyMessages(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Debugf("")
	Infof("")
	Warnf("")
	Errorf("")
	Fatalf("")

	assert.Len(t, mock.debugCalls, 1, "Should have one debug call")
	assert.Len(t, mock.infoCalls, 1, "Should have one info call")
	assert.Len(t, mock.warnCalls, 1, "Should have one warn call")
	assert.Len(t, mock.errorCalls, 1, "Should have one error call")
	assert.Len(t, mock.fatalCalls, 1, "Should have one fatal call")

	assert.Equal(t, "", mock.debugCalls[0].msg)
	assert.Equal(t, "", mock.infoCalls[0].msg)
	assert.Equal(t, "", mock.warnCalls[0].msg)
	assert.Equal(t, "", mock.errorCalls[0].msg)
	assert.Equal(t, "", mock.fatalCalls[0].msg)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestMultipleArguments(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Infof("message: %s, number: %d, float: %f, bool: %v", "test", 42, 3.14, true)

	assert.Len(t, mock.infoCalls, 1, "Should have one info call")
	assert.Equal(t, "message: %s, number: %d, float: %f, bool: %v", mock.infoCalls[0].msg)
	assert.Equal(t, []any{"test", 42, 3.14, true}, mock.infoCalls[0].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestNoArgs(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Infof("simple message")
	Debugf("simple debug")
	Warnf("simple warning")
	Errorf("simple error")
	Fatalf("simple fatal")

	assert.Len(t, mock.infoCalls, 1, "Should have one info call")
	assert.Len(t, mock.debugCalls, 1, "Should have one debug call")
	assert.Len(t, mock.warnCalls, 1, "Should have one warn call")
	assert.Len(t, mock.errorCalls, 1, "Should have one error call")
	assert.Len(t, mock.fatalCalls, 1, "Should have one fatal call")

	assert.Empty(t, mock.infoCalls[0].args)
	assert.Empty(t, mock.debugCalls[0].args)
	assert.Empty(t, mock.warnCalls[0].args)
	assert.Empty(t, mock.errorCalls[0].args)
	assert.Empty(t, mock.fatalCalls[0].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestWithRealGlogLogger(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	// Use real GlogLogger
	logger = NewGlogLogger()

	// These should not panic
	assert.NotPanics(t, func() {
		Debugf("debug message")
		Infof("info message")
		Warnf("warning message")
		Errorf("error message")
	}, "Real GlogLogger should not panic")
}

func TestSpecialCharacters(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Infof("message with special chars: \n\t\"quotes\" and 'apostrophes'")

	assert.Len(t, mock.infoCalls, 1, "Should have one info call")
	assert.Equal(t, "message with special chars: \n\t\"quotes\" and 'apostrophes'", mock.infoCalls[0].msg)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestLoggerInterfaceCompliance(t *testing.T) {
	var _ Logger = (*mockLogger)(nil)
	var _ Logger = (*GlogLogger)(nil)
}

func TestFatal(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")

	mock := newMockLogger()
	logger = mock

	Fatalf("fatal message")
	assert.Len(t, mock.fatalCalls, 1, "Should have one fatal call")
	assert.Equal(t, "fatal message", mock.fatalCalls[0].msg)
	assert.Empty(t, mock.fatalCalls[0].args)

	Fatalf("fatal with args: %s, %d", "test", 111)
	assert.Len(t, mock.fatalCalls, 2, "Should have two fatal calls")
	assert.Equal(t, "fatal with args: %s, %d", mock.fatalCalls[1].msg)
	assert.Equal(t, []any{"test", 111}, mock.fatalCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

// Tests for StructuredLogger interface methods

func TestSlogDebug(t *testing.T) {
	mock := newMockLogger()
	logger = mock

	// Test Debug (non-context variant)
	logger.Debug("debug message")
	assert.Len(t, mock.debugCalls, 1, "Should have one debug call")
	assert.Equal(t, "debug message", mock.debugCalls[0].msg)
	assert.Empty(t, mock.debugCalls[0].args)

	logger.Debug("debug with args", "key", "value", "number", 42)
	assert.Len(t, mock.debugCalls, 2, "Should have two debug calls")
	assert.Equal(t, "debug with args", mock.debugCalls[1].msg)
	assert.Equal(t, []any{"key", "value", "number", 42}, mock.debugCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogDebugContext(t *testing.T) {
	mock := newMockLogger()
	logger = mock
	ctx := context.Background()

	// Test DebugContext
	logger.DebugContext(ctx, "debug with context")
	assert.Len(t, mock.debugContextCalls, 1, "Should have one debug context call")
	assert.Equal(t, "debug with context", mock.debugContextCalls[0].msg)
	assert.Equal(t, ctx, mock.debugContextCalls[0].ctx)
	assert.Empty(t, mock.debugContextCalls[0].args)

	logger.DebugContext(ctx, "debug context with args", "key", "value")
	assert.Len(t, mock.debugContextCalls, 2, "Should have two debug context calls")
	assert.Equal(t, "debug context with args", mock.debugContextCalls[1].msg)
	assert.Equal(t, []any{"key", "value"}, mock.debugContextCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogInfo(t *testing.T) {
	mock := newMockLogger()
	logger = mock

	logger.Info("info message")
	assert.Len(t, mock.infoCalls, 1, "Should have one info call")
	assert.Equal(t, "info message", mock.infoCalls[0].msg)
	assert.Empty(t, mock.infoCalls[0].args)

	logger.Info("info with args", "status", "ok")
	assert.Len(t, mock.infoCalls, 2, "Should have two info calls")
	assert.Equal(t, "info with args", mock.infoCalls[1].msg)
	assert.Equal(t, []any{"status", "ok"}, mock.infoCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogInfoContext(t *testing.T) {
	mock := newMockLogger()
	logger = mock
	ctx := context.WithValue(context.Background(), "requestID", "12345")

	logger.InfoContext(ctx, "info with context")
	assert.Len(t, mock.infoContextCalls, 1, "Should have one info context call")
	assert.Equal(t, "info with context", mock.infoContextCalls[0].msg)
	assert.Equal(t, ctx, mock.infoContextCalls[0].ctx)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogWarn(t *testing.T) {
	mock := newMockLogger()
	logger = mock

	logger.Warn("warning message")
	assert.Len(t, mock.warnCalls, 1, "Should have one warn call")
	assert.Equal(t, "warning message", mock.warnCalls[0].msg)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogWarnContext(t *testing.T) {
	mock := newMockLogger()
	logger = mock
	ctx := context.Background()

	logger.WarnContext(ctx, "warning with context", "severity", "medium")
	assert.Len(t, mock.warnContextCalls, 1, "Should have one warn context call")
	assert.Equal(t, "warning with context", mock.warnContextCalls[0].msg)
	assert.Equal(t, []any{"severity", "medium"}, mock.warnContextCalls[0].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogError(t *testing.T) {
	mock := newMockLogger()
	logger = mock

	logger.Error("error message")
	assert.Len(t, mock.errorCalls, 1, "Should have one error call")
	assert.Equal(t, "error message", mock.errorCalls[0].msg)

	logger.Error("error with details", "code", 500, "err", "internal error")
	assert.Len(t, mock.errorCalls, 2, "Should have two error calls")
	assert.Equal(t, []any{"code", 500, "err", "internal error"}, mock.errorCalls[1].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogErrorContext(t *testing.T) {
	mock := newMockLogger()
	logger = mock
	ctx := context.Background()

	logger.ErrorContext(ctx, "error with context", "component", "api")
	assert.Len(t, mock.errorContextCalls, 1, "Should have one error context call")
	assert.Equal(t, "error with context", mock.errorContextCalls[0].msg)
	assert.Equal(t, ctx, mock.errorContextCalls[0].ctx)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogFatal(t *testing.T) {
	mock := newMockLogger()
	logger = mock

	logger.Fatal("fatal error")
	assert.Len(t, mock.fatalCalls, 1, "Should have one fatal call")
	assert.Equal(t, "fatal error", mock.fatalCalls[0].msg)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogFatalContext(t *testing.T) {
	mock := newMockLogger()
	logger = mock
	ctx := context.Background()

	logger.FatalContext(ctx, "fatal with context", "reason", "shutdown")
	assert.Len(t, mock.fatalContextCalls, 1, "Should have one fatal context call")
	assert.Equal(t, "fatal with context", mock.fatalContextCalls[0].msg)
	assert.Equal(t, []any{"reason", "shutdown"}, mock.fatalContextCalls[0].args)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestSlogAllMethods(t *testing.T) {
	mock := newMockLogger()
	logger = mock
	ctx := context.Background()

	// Test that all slog methods work without panicking
	logger.Debug("debug")
	logger.DebugContext(ctx, "debug context")
	logger.Info("info")
	logger.InfoContext(ctx, "info context")
	logger.Warn("warn")
	logger.WarnContext(ctx, "warn context")
	logger.Error("error")
	logger.ErrorContext(ctx, "error context")
	logger.Fatal("fatal")
	logger.FatalContext(ctx, "fatal context")

	// Verify all calls were recorded
	assert.Len(t, mock.debugCalls, 1)
	assert.Len(t, mock.debugContextCalls, 1)
	assert.Len(t, mock.infoCalls, 1)
	assert.Len(t, mock.infoContextCalls, 1)
	assert.Len(t, mock.warnCalls, 1)
	assert.Len(t, mock.warnContextCalls, 1)
	assert.Len(t, mock.errorCalls, 1)
	assert.Len(t, mock.errorContextCalls, 1)
	assert.Len(t, mock.fatalCalls, 1)
	assert.Len(t, mock.fatalContextCalls, 1)

	// Restore default logger
	logger = NewGlogLogger()
}

func TestWithRealGlogLoggerSlog(t *testing.T) {
	// Initialize glog flags
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	// Use real GlogLogger
	logger = NewGlogLogger()
	ctx := context.Background()

	// These should not panic
	assert.NotPanics(t, func() {
		logger.Debug("debug message")
		logger.DebugContext(ctx, "debug with context")
		logger.Info("info message")
		logger.InfoContext(ctx, "info with context")
		logger.Warn("warn message")
		logger.WarnContext(ctx, "warn with context")
		logger.Error("error message")
		logger.ErrorContext(ctx, "error with context")
	}, "Real GlogLogger slog methods should not panic")
}

func TestLevelFatalConstant(t *testing.T) {
	// Verify that LevelFatal is defined correctly
	assert.Equal(t, LevelFatal, slog.LevelError+4, "LevelFatal should be slog.LevelError + 4")
}
