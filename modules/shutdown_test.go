package modules

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockShutdownModule is a test implementation of the ShutdownModule interface
type mockShutdownModule struct {
	name          string
	shutdownCalls int
	shouldError   bool
}

func (m *mockShutdownModule) Shutdown() error {
	m.shutdownCalls++
	if m.shouldError {
		return errors.New("mock shutdown error")
	}
	return nil
}

// nonShutdownModule doesn't implement the ShutdownModule interface
type nonShutdownModule struct {
	name string
}

func TestNewShutdownModules(t *testing.T) {
	tests := []struct {
		name              string
		modules           map[string]interface{}
		expectModuleNames []string
	}{
		{
			name:              "nil-modules",
			modules:           nil,
			expectModuleNames: []string{},
		},
		{
			name:              "empty-modules",
			modules:           map[string]interface{}{},
			expectModuleNames: []string{},
		},
		{
			name: "single-module",
			modules: map[string]interface{}{
				"module1": &mockShutdownModule{name: "module1"},
			},
			expectModuleNames: []string{"module1"},
		},
		{
			name: "multiple-modules",
			modules: map[string]interface{}{
				"module1": &mockShutdownModule{name: "module1"},
				"module2": &mockShutdownModule{name: "module2"},
				"module3": &mockShutdownModule{name: "module3"},
			},
			expectModuleNames: []string{"module1", "module2", "module3"},
		},
		{
			name: "non-shutdown-module",
			modules: map[string]interface{}{
				"module1": &mockShutdownModule{name: "module1"},
				"module2": &nonShutdownModule{name: "non-shutdown-module"},
				"module3": &mockShutdownModule{name: "module3"},
			},
			expectModuleNames: []string{"module1", "module3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result := NewShutdownModules(tt.modules)

			assert.NotNil(t, result)

			resultModuleNames := make([]string, len(result.modules))
			for i, module := range result.modules {
				resultModuleNames[i] = module.(*mockShutdownModule).name
			}

			assert.ElementsMatch(t, tt.expectModuleNames, resultModuleNames)
		})
	}
}

func TestShutdownModules_Shutdown(t *testing.T) {
	tests := []struct {
		name    string
		modules []Shutdowner
	}{
		{
			name:    "empty-modules-succeeds",
			modules: []Shutdowner{},
		},
		{
			name: "single-module-success",
			modules: []Shutdowner{
				&mockShutdownModule{name: "module1", shouldError: false},
			},
		},
		{
			name: "multiple-modules-success",
			modules: []Shutdowner{
				&mockShutdownModule{name: "module1", shouldError: false},
				&mockShutdownModule{name: "module2", shouldError: false},
				&mockShutdownModule{name: "module3", shouldError: false},
			},
		},
		{
			name: "first-module-error-returns-error",
			modules: []Shutdowner{
				&mockShutdownModule{name: "module1", shouldError: true},
				&mockShutdownModule{name: "module2", shouldError: false},
			},
		},
		{
			name: "middle-module-error-returns-error",
			modules: []Shutdowner{
				&mockShutdownModule{name: "module1", shouldError: false},
				&mockShutdownModule{name: "module2", shouldError: true},
				&mockShutdownModule{name: "module3", shouldError: false},
			},
		},
		{
			name: "all-modules-called-despite-errors",
			modules: []Shutdowner{
				&mockShutdownModule{name: "module1", shouldError: true},
				&mockShutdownModule{name: "module2", shouldError: true},
				&mockShutdownModule{name: "module3", shouldError: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			sm := &ShutdownModules{
				modules: tt.modules,
			}
			sm.Shutdown()

			for _, module := range tt.modules {
				mockModule, ok := module.(*mockShutdownModule)
				assert.True(t, ok)
				assert.Equal(t, 1, mockModule.shutdownCalls)
			}
		})
	}
}
