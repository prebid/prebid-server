package logger

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestRegisterLogger(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	mockLogger := &MockLogger{}

	// Test registering a logger
	RegisterLogger("test-logger", mockLogger)

	// Verify logger was registered
	if len(manager.loggers) != 1 {
		t.Errorf("Expected 1 logger, got %d", len(manager.loggers))
	}

	if manager.loggers["test-logger"] != mockLogger {
		t.Error("Logger was not registered correctly")
	}
}

func TestRegisterLogger_Overwrite(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	mockLogger1 := &MockLogger{}
	mockLogger2 := &MockLogger{}

	// Register first logger
	RegisterLogger("test-logger", mockLogger1)

	// Register second logger with same name (should overwrite)
	RegisterLogger("test-logger", mockLogger2)

	// Verify only one logger and it's the second one
	if len(manager.loggers) != 1 {
		t.Errorf("Expected 1 logger, got %d", len(manager.loggers))
	}

	if manager.loggers["test-logger"] != mockLogger2 {
		t.Error("Logger was not overwritten correctly")
	}
}

func TestGetLoggerByName(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	mockLogger := &MockLogger{}
	RegisterLogger("test-logger", mockLogger)

	// Test getting existing logger
	logger, err := GetLoggerByName("test-logger")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if logger != mockLogger {
		t.Error("Got wrong logger")
	}

	// Test getting non-existing logger
	logger, err = GetLoggerByName("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent logger")
	}
	if logger != nil {
		t.Error("Expected nil logger for non-existent logger")
	}

	expectedErrorMsg := "logger 'non-existent' not found"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestListLoggers(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	// Test empty list
	names := ListLoggers()
	if len(names) != 0 {
		t.Errorf("Expected empty list, got %d items", len(names))
	}

	// Add some loggers
	mockLogger1 := &MockLogger{}
	mockLogger2 := &MockLogger{}
	mockLogger3 := &MockLogger{}

	RegisterLogger("logger1", mockLogger1)
	RegisterLogger("logger2", mockLogger2)
	RegisterLogger("logger3", mockLogger3)

	// Test list with loggers
	names = ListLoggers()
	if len(names) != 3 {
		t.Errorf("Expected 3 loggers, got %d", len(names))
	}

	// Sort names to ensure consistent ordering for comparison
	sort.Strings(names)
	expected := []string{"logger1", "logger2", "logger3"}
	sort.Strings(expected)

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Expected name '%s', got '%s'", expected[i], name)
		}
	}
}

func TestRemoveLogger(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	mockLogger := &MockLogger{}
	RegisterLogger("test-logger", mockLogger)

	// Test removing existing logger
	err := RemoveLogger("test-logger")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify logger was removed
	if len(manager.loggers) != 0 {
		t.Errorf("Expected 0 loggers, got %d", len(manager.loggers))
	}

	// Test removing non-existing logger
	err = RemoveLogger("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent logger")
	}

	expectedErrorMsg := "logger 'non-existent' not found"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestLoggerManager_ThreadSafety(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup

	// Test concurrent registrations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				loggerName := fmt.Sprintf("logger-%d-%d", id, j)
				mockLogger := &MockLogger{}
				RegisterLogger(loggerName, mockLogger)
			}
		}(i)
	}

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				ListLoggers()
				loggerName := fmt.Sprintf("logger-%d-%d", id, j)
				GetLoggerByName(loggerName)
			}
		}(i)
	}

	// Test concurrent removals
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				loggerName := fmt.Sprintf("logger-%d-%d", id, j)
				RemoveLogger(loggerName)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Test should complete without deadlocks or race conditions
	// If we get here, the test passed
}

func TestLoggerManager_ConcurrentReadWrite(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	const duration = 100 * time.Millisecond
	const loggerName = "test-logger"

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		mockLogger := &MockLogger{}
		for {
			select {
			case <-done:
				return
			default:
				RegisterLogger(loggerName, mockLogger)
				RemoveLogger(loggerName)
			}
		}
	}()

	// Reader goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					GetLoggerByName(loggerName)
					ListLoggers()
				}
			}
		}()
	}

	// Let it run for a bit
	time.Sleep(duration)
	close(done)

	// Wait for all goroutines to complete
	wg.Wait()

	// Test should complete without deadlocks or race conditions
}

func TestLoggerManager_EdgeCases(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	// Test empty string logger name
	mockLogger := &MockLogger{}
	RegisterLogger("", mockLogger)

	logger, err := GetLoggerByName("")
	if err != nil {
		t.Errorf("Expected no error for empty string logger name, got %v", err)
	}
	if logger != mockLogger {
		t.Error("Empty string logger name should work")
	}

	// Test removing empty string logger
	err = RemoveLogger("")
	if err != nil {
		t.Errorf("Expected no error removing empty string logger, got %v", err)
	}

	// Test very long logger name
	longName := string(make([]byte, 10000))
	for i := range longName {
		longName = longName[:i] + "a" + longName[i+1:]
	}

	RegisterLogger(longName, mockLogger)

	logger, err = GetLoggerByName(longName)
	if err != nil {
		t.Errorf("Expected no error for long logger name, got %v", err)
	}
	if logger != mockLogger {
		t.Error("Long logger name should work")
	}
}

func TestLoggerManager_NilLogger(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	// Test registering nil logger
	RegisterLogger("nil-logger", nil)

	// Verify nil logger was registered
	if len(manager.loggers) != 1 {
		t.Errorf("Expected 1 logger, got %d", len(manager.loggers))
	}

	logger, err := GetLoggerByName("nil-logger")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if logger != nil {
		t.Error("Expected nil logger")
	}
}

func TestLoggerManager_MultipleOperations(t *testing.T) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	// Test sequence of operations
	mockLogger1 := &MockLogger{}
	mockLogger2 := &MockLogger{}
	mockLogger3 := &MockLogger{}

	// Register multiple loggers
	RegisterLogger("logger1", mockLogger1)
	RegisterLogger("logger2", mockLogger2)
	RegisterLogger("logger3", mockLogger3)

	// Verify all are registered
	names := ListLoggers()
	if len(names) != 3 {
		t.Errorf("Expected 3 loggers, got %d", len(names))
	}

	// Remove one logger
	err := RemoveLogger("logger2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify removal
	names = ListLoggers()
	if len(names) != 2 {
		t.Errorf("Expected 2 loggers after removal, got %d", len(names))
	}

	// Verify removed logger is not accessible
	_, err = GetLoggerByName("logger2")
	if err == nil {
		t.Error("Expected error for removed logger")
	}

	// Verify other loggers are still accessible
	logger, err := GetLoggerByName("logger1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if logger != mockLogger1 {
		t.Error("Got wrong logger")
	}

	logger, err = GetLoggerByName("logger3")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if logger != mockLogger3 {
		t.Error("Got wrong logger")
	}
}

// Benchmark tests
func BenchmarkRegisterLogger(b *testing.B) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	mockLogger := &MockLogger{}
	for i := 0; i < b.N; i++ {
		RegisterLogger(fmt.Sprintf("logger-%d", i), mockLogger)
	}
}

func BenchmarkGetLoggerByName(b *testing.B) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	mockLogger := &MockLogger{}
	RegisterLogger("test-logger", mockLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetLoggerByName("test-logger")
	}
}

func BenchmarkListLoggers(b *testing.B) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	// Add some loggers
	mockLogger := &MockLogger{}
	for i := 0; i < 100; i++ {
		RegisterLogger(fmt.Sprintf("logger-%d", i), mockLogger)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ListLoggers()
	}
}

func BenchmarkRemoveLogger(b *testing.B) {
	// Save original manager state
	originalManager := manager
	defer func() {
		manager = originalManager
	}()

	// Create fresh manager for test
	manager = &LoggerManager{
		loggers: make(map[string]Logger),
	}

	mockLogger := &MockLogger{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		loggerName := fmt.Sprintf("logger-%d", i)
		RegisterLogger(loggerName, mockLogger)
		b.StartTimer()

		RemoveLogger(loggerName)
	}
}
