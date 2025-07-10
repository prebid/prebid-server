package logger

import (
	"fmt"
	"sync"
)

// LoggerManager manages multiple logger instances
type LoggerManager struct {
	loggers map[string]Logger
	mu      sync.RWMutex
}

var manager = &LoggerManager{
	loggers: make(map[string]Logger),
}

// RegisterLogger registers a logger with a specific name
func RegisterLogger(name string, logger Logger) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.loggers[name] = logger
}

// GetLoggerByName returns a logger by name
func GetLoggerByName(name string) (Logger, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	if logger, exists := manager.loggers[name]; exists {
		return logger, nil
	}
	return nil, fmt.Errorf("logger '%s' not found", name)
}

// ListLoggers returns all registered logger names
func ListLoggers() []string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	names := make([]string, 0, len(manager.loggers))
	for name := range manager.loggers {
		names = append(names, name)
	}
	return names
}

// RemoveLogger removes a logger by name
func RemoveLogger(name string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, exists := manager.loggers[name]; !exists {
		return fmt.Errorf("logger '%s' not found", name)
	}

	delete(manager.loggers, name)
	return nil
}
