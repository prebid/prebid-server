package modules

import (
	"github.com/golang/glog"
)

// Shutdowner is an interface that defines a method for shutting down modules.
type Shutdowner interface {
	Shutdown() error
}

// ShutdownModules is a struct that holds a slice of Shutdowner modules.
type ShutdownModules struct {
	modules []Shutdowner
}

// NewShutdownModules creates a new ShutdownModules instance from a map of modules.
// It filters the modules to include only those that implement the Shutdowner interface.
func NewShutdownModules(modules map[string]interface{}) *ShutdownModules {
	sm := ShutdownModules{
		modules: make([]Shutdowner, 0),
	}

	for _, module := range modules {
		if v, ok := module.(Shutdowner); ok {
			sm.modules = append(sm.modules, v)
		}
	}
	return &sm
}

// Shutdown iterates over all modules and calls their Shutdown method.
func (s *ShutdownModules) Shutdown() {
	for _, module := range s.modules {
		if err := module.Shutdown(); err != nil {
			glog.Errorf("Error shutting down module: %v", err)
		}
	}
	return
}
