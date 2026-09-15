package logger

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockLogger is a test implementation of the Logger interface
type mockLogger struct {
	debugCalls []logCall
	infoCalls  []logCall
	warnCalls  []logCall
	errorCalls []logCall
	fatalCalls []logCall
}

type logCall struct {
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

func newMockLogger() *mockLogger {
	return &mockLogger{
		debugCalls: []logCall{},
		infoCalls:  []logCall{},
		warnCalls:  []logCall{},
		errorCalls: []logCall{},
		fatalCalls: []logCall{},
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
