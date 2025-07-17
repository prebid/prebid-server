package logger

import (
	"testing"
)

// MockLogger is a mock implementation of the Logger interface for testing
type MockLogger struct {
	InfoCalls     [][]any
	InfofCalls    []InfofCall
	WarningCalls  [][]any
	WarningfCalls []WarningfCall
	ErrorCalls    [][]any
	ErrorfCalls   []ErrorfCall
	ExitfCalls    []ExitfCall
	FatalCalls    [][]any
	FatalfCalls   []FatalfCall
}

type InfofCall struct {
	Format string
	Args   []any
}

type WarningfCall struct {
	Format string
	Args   []any
}

type ErrorfCall struct {
	Format string
	Args   []any
}

type ExitfCall struct {
	Format string
	Args   []any
}

type FatalfCall struct {
	Format string
	Args   []any
}

func (m *MockLogger) Info(args ...any) {
	m.InfoCalls = append(m.InfoCalls, args)
}

func (m *MockLogger) Infof(format string, args ...any) {
	m.InfofCalls = append(m.InfofCalls, InfofCall{Format: format, Args: args})
}

func (m *MockLogger) Warning(args ...any) {
	m.WarningCalls = append(m.WarningCalls, args)
}

func (m *MockLogger) Warningf(format string, args ...any) {
	m.WarningfCalls = append(m.WarningfCalls, WarningfCall{Format: format, Args: args})
}

func (m *MockLogger) Error(args ...any) {
	m.ErrorCalls = append(m.ErrorCalls, args)
}

func (m *MockLogger) Errorf(format string, args ...any) {
	m.ErrorfCalls = append(m.ErrorfCalls, ErrorfCall{Format: format, Args: args})
}

func (m *MockLogger) Exitf(format string, args ...any) {
	m.ExitfCalls = append(m.ExitfCalls, ExitfCall{Format: format, Args: args})
}

func (m *MockLogger) Fatal(args ...any) {
	m.FatalCalls = append(m.FatalCalls, args)
}

func (m *MockLogger) Fatalf(format string, args ...any) {
	m.FatalfCalls = append(m.FatalfCalls, FatalfCall{Format: format, Args: args})
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		loggerType  string
		depth       *int
		expectedErr bool
	}{
		{
			name:        "Valid default logger with nil depth",
			loggerType:  "default",
			depth:       nil,
			expectedErr: false,
		},
		{
			name:        "Valid default logger with valid depth",
			loggerType:  "default",
			depth:       intPtr(3),
			expectedErr: false,
		},
		{
			name:        "Valid default logger with zero depth",
			loggerType:  "default",
			depth:       intPtr(0),
			expectedErr: false,
		},
		{
			name:        "Valid default logger with max depth",
			loggerType:  "default",
			depth:       intPtr(10),
			expectedErr: false,
		},
		{
			name:        "Invalid logger type",
			loggerType:  "invalid",
			depth:       nil,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.loggerType, tt.depth)
			if (err != nil) != tt.expectedErr {
				t.Errorf("New() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestNewWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *LoggerConfig
		expectedErr bool
	}{
		{
			name: "Valid config with default logger",
			config: &LoggerConfig{
				Type:  LoggerTypeDefault,
				Depth: intPtr(2),
			},
			expectedErr: false,
		},
		{
			name: "Valid config with nil depth",
			config: &LoggerConfig{
				Type:  LoggerTypeDefault,
				Depth: nil,
			},
			expectedErr: false,
		},
		{
			name: "Valid config with negative depth (should use default)",
			config: &LoggerConfig{
				Type:  LoggerTypeDefault,
				Depth: intPtr(-1),
			},
			expectedErr: false,
		},
		{
			name: "Valid config with depth > 10 (should use default)",
			config: &LoggerConfig{
				Type:  LoggerTypeDefault,
				Depth: intPtr(11),
			},
			expectedErr: false,
		},
		{
			name: "Invalid logger type",
			config: &LoggerConfig{
				Type:  "invalid",
				Depth: intPtr(2),
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewWithConfig(tt.config)
			if (err != nil) != tt.expectedErr {
				t.Errorf("NewWithConfig() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestNewWithConfig_ErrorMessage(t *testing.T) {
	config := &LoggerConfig{
		Type:  "unsupported",
		Depth: intPtr(1),
	}

	err := NewWithConfig(config)
	if err == nil {
		t.Fatal("Expected error for unsupported logger type")
	}

	expectedMsg := "unsupported logger type: unsupported"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestLoggerMethods(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Set up mock logger
	mockLogger := &MockLogger{}
	logger = mockLogger

	// Test Info
	Info("test", "info")
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}
	if len(mockLogger.InfoCalls[0]) != 2 || mockLogger.InfoCalls[0][0] != "test" || mockLogger.InfoCalls[0][1] != "info" {
		t.Errorf("Info args not passed correctly: %v", mockLogger.InfoCalls[0])
	}

	// Test Infof
	Infof("test %s", "info")
	if len(mockLogger.InfofCalls) != 1 {
		t.Errorf("Expected 1 Infof call, got %d", len(mockLogger.InfofCalls))
	}

	if mockLogger.InfofCalls[0].Format != "test %s" || len(mockLogger.InfofCalls[0].Args) != 1 || mockLogger.InfofCalls[0].Args[0].([]any)[0] != "info" {
		t.Errorf("Infof args not passed correctly: %v", mockLogger.InfofCalls[0])
	}

	// Test Warning
	Warning("test", "warning")
	if len(mockLogger.WarningCalls) != 1 {
		t.Errorf("Expected 1 Warning call, got %d", len(mockLogger.WarningCalls))
	}
	if len(mockLogger.WarningCalls[0]) != 2 || mockLogger.WarningCalls[0][0] != "test" || mockLogger.WarningCalls[0][1] != "warning" {
		t.Errorf("Warning args not passed correctly: %v", mockLogger.WarningCalls[0])
	}

	// Test Warningf
	Warningf("test %s", "warning")
	if len(mockLogger.WarningfCalls) != 1 {
		t.Errorf("Expected 1 Warningf call, got %d", len(mockLogger.WarningfCalls))
	}
	if mockLogger.WarningfCalls[0].Format != "test %s" || len(mockLogger.WarningfCalls[0].Args) != 1 || mockLogger.WarningfCalls[0].Args[0].([]any)[0] != "warning" {
		t.Errorf("Warningf args not passed correctly: %v", mockLogger.WarningfCalls[0])
	}

	// Test Error
	Error("test", "error")
	if len(mockLogger.ErrorCalls) != 1 {
		t.Errorf("Expected 1 Error call, got %d", len(mockLogger.ErrorCalls))
	}
	if len(mockLogger.ErrorCalls[0]) != 2 || mockLogger.ErrorCalls[0][0] != "test" || mockLogger.ErrorCalls[0][1] != "error" {
		t.Errorf("Error args not passed correctly: %v", mockLogger.ErrorCalls[0])
	}

	// Test Errorf
	Errorf("test %s", "error")
	if len(mockLogger.ErrorfCalls) != 1 {
		t.Errorf("Expected 1 Errorf call, got %d", len(mockLogger.ErrorfCalls))
	}
	if mockLogger.ErrorfCalls[0].Format != "test %s" || len(mockLogger.ErrorfCalls[0].Args) != 1 || mockLogger.ErrorfCalls[0].Args[0].([]any)[0] != "error" {
		t.Errorf("Errorf args not passed correctly: %v", mockLogger.ErrorfCalls[0])
	}

	// Test Exitf
	Exitf("test %s", "exit")
	if len(mockLogger.ExitfCalls) != 1 {
		t.Errorf("Expected 1 Exitf call, got %d", len(mockLogger.ExitfCalls))
	}
	if mockLogger.ExitfCalls[0].Format != "test %s" || len(mockLogger.ExitfCalls[0].Args) != 1 || mockLogger.ExitfCalls[0].Args[0].([]any)[0] != "exit" {
		t.Errorf("Exitf args not passed correctly: %v", mockLogger.ExitfCalls[0])
	}

	// Test Fatal
	Fatal("test", "fatal")
	if len(mockLogger.FatalCalls) != 1 {
		t.Errorf("Expected 1 Fatal call, got %d", len(mockLogger.FatalCalls))
	}
	if len(mockLogger.FatalCalls[0]) != 2 || mockLogger.FatalCalls[0][0] != "test" || mockLogger.FatalCalls[0][1] != "fatal" {
		t.Errorf("Fatal args not passed correctly: %v", mockLogger.FatalCalls[0])
	}

	// Test Fatalf
	Fatalf("test %s", "fatal")
	if len(mockLogger.FatalfCalls) != 1 {
		t.Errorf("Expected 1 Fatalf call, got %d", len(mockLogger.FatalfCalls))
	}
	if mockLogger.FatalfCalls[0].Format != "test %s" || len(mockLogger.FatalfCalls[0].Args) != 1 || mockLogger.FatalfCalls[0].Args[0].([]any)[0] != "fatal" {
		t.Errorf("Fatalf args not passed correctly: %v", mockLogger.FatalfCalls[0])
	}
}

func TestLoggerType(t *testing.T) {
	if LoggerTypeDefault != "default" {
		t.Errorf("Expected LoggerTypeDefault to be 'default', got '%s'", LoggerTypeDefault)
	}
}

func TestLoggerConfig(t *testing.T) {
	depth := 5
	config := &LoggerConfig{
		Type:  LoggerTypeDefault,
		Depth: &depth,
	}

	if config.Type != LoggerTypeDefault {
		t.Errorf("Expected Type to be %s, got %s", LoggerTypeDefault, config.Type)
	}

	if config.Depth == nil || *config.Depth != 5 {
		t.Errorf("Expected Depth to be 5, got %v", config.Depth)
	}
}

func TestDepthValidation(t *testing.T) {
	tests := []struct {
		name             string
		inputDepth       *int
		shouldUseDefault bool
	}{
		{
			name:             "Nil depth should use default",
			inputDepth:       nil,
			shouldUseDefault: true,
		},
		{
			name:             "Valid depth should be used",
			inputDepth:       intPtr(5),
			shouldUseDefault: false,
		},
		{
			name:             "Zero depth should be valid",
			inputDepth:       intPtr(0),
			shouldUseDefault: false,
		},
		{
			name:             "Max depth should be valid",
			inputDepth:       intPtr(10),
			shouldUseDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LoggerConfig{
				Type:  LoggerTypeDefault,
				Depth: tt.inputDepth,
			}

			err := NewWithConfig(config)
			if err != nil {
				t.Errorf("NewWithConfig() returned error: %v", err)
			}

			if tt.shouldUseDefault {
				if config.Depth == nil || *config.Depth != defaultDepth {
					t.Errorf("Expected depth to be reset to default (%d), got %v", defaultDepth, config.Depth)
				}
			} else {
				if config.Depth == nil || *config.Depth != *tt.inputDepth {
					t.Errorf("Expected depth to remain %d, got %v", *tt.inputDepth, config.Depth)
				}
			}
		})
	}
}

func TestDefaultDepth(t *testing.T) {
	if defaultDepth != 1 {
		t.Errorf("Expected defaultDepth to be 1, got %d", defaultDepth)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Benchmark tests
func BenchmarkNew(b *testing.B) {
	depth := 2
	for i := 0; i < b.N; i++ {
		New("default", &depth)
	}
}

func BenchmarkNewWithConfig(b *testing.B) {
	config := &LoggerConfig{
		Type:  LoggerTypeDefault,
		Depth: intPtr(2),
	}
	for i := 0; i < b.N; i++ {
		NewWithConfig(config)
	}
}

func BenchmarkInfo(b *testing.B) {
	mockLogger := &MockLogger{}
	originalLogger := logger
	logger = mockLogger
	defer func() {
		logger = originalLogger
	}()

	for i := 0; i < b.N; i++ {
		Info("test message")
	}
}

func BenchmarkInfof(b *testing.B) {
	mockLogger := &MockLogger{}
	originalLogger := logger
	logger = mockLogger
	defer func() {
		logger = originalLogger
	}()

	for i := 0; i < b.N; i++ {
		Infof("test message %d", i)
	}
}

func TestSetCustomLogger(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Create mock logger
	mockLogger := &MockLogger{}

	// Test SetCustomLogger
	SetCustomLogger(mockLogger)

	// Verify the logger was set
	if logger != mockLogger {
		t.Error("SetCustomLogger did not set the logger correctly")
	}

	// Test that the logger works
	Info("test message")
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}
	if len(mockLogger.InfoCalls[0]) != 1 || mockLogger.InfoCalls[0][0] != "test message" {
		t.Errorf("Info call arguments incorrect: %v", mockLogger.InfoCalls[0])
	}
}

func TestSetCustomLoggerByName(t *testing.T) {
	// Save original logger and clear manager
	originalLogger := logger
	originalLoggers := manager.loggers
	defer func() {
		logger = originalLogger
		manager.loggers = originalLoggers
	}()

	// Reset manager
	manager.loggers = make(map[string]Logger)

	// Create and register mock logger
	mockLogger := &MockLogger{}
	RegisterLogger("test-logger", mockLogger)

	// Test SetCustomLoggerByName with existing logger
	err := SetCustomLoggerByName("test-logger")
	if err != nil {
		t.Errorf("SetCustomLoggerByName returned error: %v", err)
	}

	// Verify the logger was set
	if logger != mockLogger {
		t.Error("SetCustomLoggerByName did not set the logger correctly")
	}

	// Test that the logger works
	Info("test message")
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}

	// Test SetCustomLoggerByName with non-existing logger
	err = SetCustomLoggerByName("non-existing-logger")
	if err == nil {
		t.Error("SetCustomLoggerByName should return error for non-existing logger")
	}

	expectedError := "failed to set custom logger 'non-existing-logger': logger 'non-existing-logger' not found"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetCurrentLogger(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Test with default logger
	defaultLogger := NewDefaultLogger(1)
	logger = defaultLogger

	currentLogger := GetCurrentLogger()
	if currentLogger != defaultLogger {
		t.Error("GetCurrentLogger did not return the correct logger")
	}

	// Test with custom logger
	mockLogger := &MockLogger{}
	logger = mockLogger

	currentLogger = GetCurrentLogger()
	if currentLogger != mockLogger {
		t.Error("GetCurrentLogger did not return the correct custom logger")
	}
}

func TestNewCustom(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Create mock logger
	mockLogger := &MockLogger{}

	// Test NewCustom
	err := NewCustom(mockLogger)
	if err != nil {
		t.Errorf("NewCustom returned error: %v", err)
	}

	// Verify the logger was set
	if logger != mockLogger {
		t.Error("NewCustom did not set the logger correctly")
	}

	// Test that the logger works
	Info("test message")
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}
}

func TestNewCustomByName(t *testing.T) {
	// Save original logger and clear manager
	originalLogger := logger
	originalLoggers := manager.loggers
	defer func() {
		logger = originalLogger
		manager.loggers = originalLoggers
	}()

	// Reset manager
	manager.loggers = make(map[string]Logger)

	// Create and register mock logger
	mockLogger := &MockLogger{}
	RegisterLogger("test-logger", mockLogger)

	// Test NewCustomByName with existing logger
	err := NewCustomByName("test-logger")
	if err != nil {
		t.Errorf("NewCustomByName returned error: %v", err)
	}

	// Verify the logger was set
	if logger != mockLogger {
		t.Error("NewCustomByName did not set the logger correctly")
	}

	// Test that the logger works
	Info("test message")
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}

	// Test NewCustomByName with non-existing logger
	err = NewCustomByName("non-existing-logger")
	if err == nil {
		t.Error("NewCustomByName should return error for non-existing logger")
	}

	expectedError := "failed to get custom logger 'non-existing-logger': logger 'non-existing-logger' not found"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestNewWithConfig_CustomLogger(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Test custom logger with instance
	mockLogger := &MockLogger{}
	config := &LoggerConfig{
		Type:         LoggerTypeCustom,
		CustomLogger: mockLogger,
	}

	err := NewWithConfig(config)
	if err != nil {
		t.Errorf("NewWithConfig returned error: %v", err)
	}

	if logger != mockLogger {
		t.Error("NewWithConfig did not set the custom logger correctly")
	}

	// Test that the logger works
	Info("test message")
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}
}

func TestNewWithConfig_CustomLoggerByName(t *testing.T) {
	// Save original logger and clear manager
	originalLogger := logger
	originalLoggers := manager.loggers
	defer func() {
		logger = originalLogger
		manager.loggers = originalLoggers
	}()

	// Reset manager
	manager.loggers = make(map[string]Logger)

	// Create and register mock logger
	mockLogger := &MockLogger{}
	RegisterLogger("test-logger", mockLogger)

	// Test custom logger with registered name
	config := &LoggerConfig{
		Type:       LoggerTypeCustom,
		CustomName: "test-logger",
	}

	err := NewWithConfig(config)
	if err != nil {
		t.Errorf("NewWithConfig returned error: %v", err)
	}

	if logger != mockLogger {
		t.Error("NewWithConfig did not set the custom logger by name correctly")
	}

	// Test that the logger works
	Info("test message")
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}
}

func TestNewWithConfig_CustomLoggerErrors(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	tests := []struct {
		name          string
		config        *LoggerConfig
		expectedError string
	}{
		{
			name: "Custom logger without instance or name",
			config: &LoggerConfig{
				Type: LoggerTypeCustom,
			},
			expectedError: "custom logger type requires either CustomLogger instance or CustomName",
		},
		{
			name: "Custom logger with non-existing name",
			config: &LoggerConfig{
				Type:       LoggerTypeCustom,
				CustomName: "non-existing-logger",
			},
			expectedError: "failed to get custom logger 'non-existing-logger': logger 'non-existing-logger' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewWithConfig(tt.config)
			if err == nil {
				t.Error("NewWithConfig should return error")
			}
			if err.Error() != tt.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestNew_CustomLoggerType(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() {
		logger = originalLogger
	}()

	// Test calling New with "custom" type should fail without proper setup
	err := New("custom", nil)
	if err == nil {
		t.Error("New with 'custom' type should return error without proper setup")
	}

	expectedError := "custom logger type requires either CustomLogger instance or CustomName"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestCustomLoggerIntegration(t *testing.T) {
	// Save original logger and clear manager
	originalLogger := logger
	originalLoggers := manager.loggers
	defer func() {
		logger = originalLogger
		manager.loggers = originalLoggers
	}()

	// Reset manager
	manager.loggers = make(map[string]Logger)

	// Create mock logger
	mockLogger := &MockLogger{}

	// Test full integration: register, set, and use
	RegisterLogger("integration-test", mockLogger)

	err := SetCustomLoggerByName("integration-test")
	if err != nil {
		t.Errorf("SetCustomLoggerByName returned error: %v", err)
	}

	// Test all logger methods
	Info("info message")
	Infof("info format %s", "test")
	Warning("warning message")
	Warningf("warning format %s", "test")
	Error("error message")
	Errorf("error format %s", "test")
	Fatal("fatal message")
	Fatalf("fatal format %s", "test")
	Exitf("exit format %s", "test")

	// Verify all calls were made
	if len(mockLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 Info call, got %d", len(mockLogger.InfoCalls))
	}
	if len(mockLogger.InfofCalls) != 1 {
		t.Errorf("Expected 1 Infof call, got %d", len(mockLogger.InfofCalls))
	}
	if len(mockLogger.WarningCalls) != 1 {
		t.Errorf("Expected 1 Warning call, got %d", len(mockLogger.WarningCalls))
	}
	if len(mockLogger.WarningfCalls) != 1 {
		t.Errorf("Expected 1 Warningf call, got %d", len(mockLogger.WarningfCalls))
	}
	if len(mockLogger.ErrorCalls) != 1 {
		t.Errorf("Expected 1 Error call, got %d", len(mockLogger.ErrorCalls))
	}
	if len(mockLogger.ErrorfCalls) != 1 {
		t.Errorf("Expected 1 Errorf call, got %d", len(mockLogger.ErrorfCalls))
	}
	if len(mockLogger.FatalCalls) != 1 {
		t.Errorf("Expected 1 Fatal call, got %d", len(mockLogger.FatalCalls))
	}
	if len(mockLogger.FatalfCalls) != 1 {
		t.Errorf("Expected 1 Fatalf call, got %d", len(mockLogger.FatalfCalls))
	}
	if len(mockLogger.ExitfCalls) != 1 {
		t.Errorf("Expected 1 Exitf call, got %d", len(mockLogger.ExitfCalls))
	}
}
