package config

type Hooks struct {
	Enabled           bool              `mapstructure:"enabled"`
	Modules           Modules           `mapstructure:"modules"`
	HostExecutionPlan HookExecutionPlan `mapstructure:"host_execution_plan"`
	// AccountExecutionPlan can be replaced by the account-specific hook execution plan
	AccountExecutionPlan HookExecutionPlan `mapstructure:"default_account_execution_plan"`
}

// Modules mapping provides module specific configuration, format: map[vendor_name]map[module_name]interface{}
// actual configuration parsing performed by modules
type Modules map[string]map[string]interface{}

type HookExecutionPlan struct {
	Endpoints map[string]struct {
		Stages map[string]struct {
			Groups []HookExecutionGroup `mapstructure:"groups" json:"groups"`
		} `mapstructure:"stages" json:"stages"`
	} `mapstructure:"endpoints" json:"endpoints"`
}

type HookExecutionGroup struct {
	// Timeout specified in milliseconds
	Timeout      int `mapstructure:"timeout" json:"timeout"`
	HookSequence []struct {
		// Module is a composite value in the format: {vendor_name}.{module_name}
		Module string `mapstructure:"module_code" json:"module_code"`
		// Hook is an arbitrary value, used to identify hook when sending metrics, storing debug information, etc.
		Hook string `mapstructure:"hook_impl_code" json:"hook_impl_code"`
	} `mapstructure:"hook_sequence" json:"hook_sequence"`
}
