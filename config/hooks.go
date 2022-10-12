package config

type Hooks struct {
	HostExecutionPlan HookExecutionPlan `mapstructure:"host-execution-plan"`
	// DefaultAccountExecutionPlan can be replaced by the account-specific hook execution plan
	DefaultAccountExecutionPlan HookExecutionPlan `mapstructure:"default-account-execution-plan"`
}

type HookExecutionPlan struct {
	Endpoints map[string]struct {
		Stages map[string]struct {
			Groups []struct {
				Timeout      int `mapstructure:"timeout" json:"timeout"`
				HookSequence []struct {
					Module string `mapstructure:"module-code" json:"module-code"`
					Hook   string `mapstructure:"hook-impl-code" json:"hook-impl-code"`
				} `mapstructure:"hook-sequence" json:"hook-sequence"`
			} `mapstructure:"groups" json:"groups"`
		} `mapstructure:"stages" json:"stages"`
	} `mapstructure:"endpoints" json:"endpoints"`
}
