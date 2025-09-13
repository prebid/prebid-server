package analytics

import "sync"

// Registry builders: vendor -> module -> builder
var (
	mu       sync.RWMutex
	registry AnalyticsModuleBuilders = make(AnalyticsModuleBuilders)
)

// Register registers a module under the same vendor and module name (e.g., "agma").
func Register(name string, b AnalyticsModuleBuilderFn) {
	RegisterVendorModule(name, name, b)
}

// RegisterVendorModule allows registering a builder with vendor/module distinction.
func RegisterVendorModule(vendor, module string, b AnalyticsModuleBuilderFn) {
	mu.Lock()
	defer mu.Unlock()
	if registry[vendor] == nil {
		registry[vendor] = make(map[string]AnalyticsModuleBuilderFn)
	}
	registry[vendor][module] = b
}

// builders returns a copy of the registered builders (for modules.go: NewBuilder()).
func builders() AnalyticsModuleBuilders {
	mu.RLock()
	defer mu.RUnlock()
	out := make(AnalyticsModuleBuilders, len(registry))
	for vendor, mods := range registry {
		cp := make(map[string]AnalyticsModuleBuilderFn, len(mods))
		for name, fn := range mods {
			cp[name] = fn
		}
		out[vendor] = cp
	}
	return out
}

func Builders() AnalyticsModuleBuilders {
	return builders()
}
