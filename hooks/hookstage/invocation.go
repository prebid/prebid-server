package hookstage

import (
	"encoding/json"
	"sync"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
)

// HookResult represents the result of execution the concrete hook instance.
type HookResult[T any] struct {
	Reject        bool         // true value indicates rejection of the program execution at the specific stage
	NbrCode       int          // hook must provide NbrCode if the field Reject set to true
	Message       string       // holds arbitrary message added by hook
	ChangeSet     ChangeSet[T] // set of changes the module wants to apply to hook payload in case of successful execution
	Errors        []string
	Warnings      []string
	DebugMessages []string
	AnalyticsTags hookanalytics.Analytics
	ModuleContext *ModuleContext // holds values that the module wants to pass to itself at later stages
}

// ModuleInvocationContext holds data passed to the module hook during invocation.
type ModuleInvocationContext struct {
	// AccountID holds the account ID
	AccountID string
	// AccountConfig represents module config rewritten at the account-level.
	AccountConfig json.RawMessage
	// Endpoint represents the path of the current endpoint.
	Endpoint string
	// ModuleContext holds values that the module passes to itself from the previous stages.
	ModuleContext *ModuleContext
	// HookImplCode is the hook_impl_code for a module instance to differentiate between multiple hooks
	HookImplCode string
}

// ModuleContext holds arbitrary data passed between module hooks at different stages.
// We use interface as we do not know exactly how the modules will use their inner context.
type ModuleContext struct {
	sync.RWMutex
	data map[string]any
}

// NewModuleContext creates a new module context
func NewModuleContext() *ModuleContext {
	moduleContext := ModuleContext{
		data: make(map[string]any),
	}
	return &moduleContext
}

// Get retrieves a value from the module context with read lock
func (mc *ModuleContext) Get(key string) (any, bool) {
	if mc == nil || mc.data == nil {
		return nil, false
	}
	mc.RLock()
	defer mc.RUnlock()
	val, ok := mc.data[key]
	return val, ok
}

// Set stores a value in the module context with write lock
func (mc *ModuleContext) Set(key string, value any) {
	if mc == nil {
		return
	}
	mc.Lock()
	defer mc.Unlock()
	if mc.data == nil {
		mc.data = make(map[string]any)
	}
	mc.data[key] = value
}

// GetAll returns a copy of all data in the context
func (mc *ModuleContext) GetAll() map[string]any {
	if mc == nil || mc.data == nil {
		return nil
	}
	mc.RLock()
	defer mc.RUnlock()
	result := make(map[string]any, len(mc.data))
	for k, v := range mc.data {
		result[k] = v
	}
	return result
}

// SetAll replaces all data in the context
func (mc *ModuleContext) SetAll(data map[string]any) {
	if mc == nil {
		return
	}
	mc.Lock()
	defer mc.Unlock()
	if mc.data == nil {
		mc.data = make(map[string]any)
	}
	for k, v := range data {
		mc.data[k] = v
	}
}
