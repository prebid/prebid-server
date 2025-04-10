package config

type Hooks struct {
	Enabled bool    `mapstructure:"enabled"`
	Modules Modules `mapstructure:"modules"`
	// HostExecutionPlan defined by the host company and is executed always
	HostExecutionPlan HookExecutionPlan `mapstructure:"host_execution_plan"`
	// DefaultAccountExecutionPlan can be replaced by the account-specific hook execution plan
	DefaultAccountExecutionPlan HookExecutionPlan `mapstructure:"default_account_execution_plan"`
}

// Modules mapping provides module specific configuration, format: map[vendor_name]map[module_name]interface{}
// actual configuration parsing performed by modules
type Modules map[string]map[string]interface{}

type HookExecutionPlan struct {
	ABTests   []ABTest `mapstructure:"abtests" json:"abtests"`
	Endpoints map[string]struct {
		Stages map[string]struct {
			Groups []HookExecutionGroup `mapstructure:"groups" json:"groups"`
		} `mapstructure:"stages" json:"stages"`
	} `mapstructure:"endpoints" json:"endpoints"`
}

type HookExecutionGroup struct {
	// Timeout specified in milliseconds.
	// Zero value marks the hook execution status with the "timeout" value.
	Timeout      int `mapstructure:"timeout" json:"timeout"`
	HookSequence []struct {
		// ModuleCode is a composite value in the format: {vendor_name}.{module_name}
		ModuleCode string `mapstructure:"module_code" json:"module_code"`
		// HookImplCode is an arbitrary value, used to identify hook when sending metrics, debug information, etc.
		HookImplCode string `mapstructure:"hook_impl_code" json:"hook_impl_code"`
	} `mapstructure:"hook_sequence" json:"hook_sequence"`
}

type ABTest struct {
	// ModuleCode is a composite value in the format: {vendor_name}.{module_name}
	ModuleCode string `mapstructure:"module_code" json:"module_code"`
	Enabled    *bool  `mapstructure:"enabled" json:"enabled"`
	// Accounts is a slice of accounts that will trigger module execution
	// An empty slice will trigger module execution for all accounts
	Accounts []interface{} `mapstructure:"accounts" json:"accounts"`
	// PercentActive enables specifying the percentage of requests that will trigger module execution
	// The default value (nil) will trigger module execution on every request, the same as if the value "100" is set
	PercentActive *uint16 `mapstructure:"percent_active" json:"percent_active"`
	// LogAnalyticsTag specifies whether module execution result will get noted in the analytics log
	// The default value (nil) will enable writing results to the analytics log, the same as if the value "true" is set
	LogAnalyticsTag *bool `mapstructure:"log_analytics_tag" json:"log_analytics_tag"`
}
