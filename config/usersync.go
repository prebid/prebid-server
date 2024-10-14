package config

// UserSync specifies the static global user sync configuration.
type UserSync struct {
	Cooperative    UserSyncCooperative `mapstructure:"coop_sync"`
	ExternalURL    string              `mapstructure:"external_url"`
	RedirectURL    string              `mapstructure:"redirect_url"`
	PriorityGroups [][]string          `mapstructure:"priority_groups"`
}

// UserSyncCooperative specifies the static global default cooperative cookie sync
type UserSyncCooperative struct {
	EnabledByDefault bool `mapstructure:"default"`
}
