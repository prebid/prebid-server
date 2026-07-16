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

// CookieSync specifies host-level cookie sync settings that are always enforced by the
// host operator. These settings cannot be overridden by account configuration and are
// unioned with any account-level restrictions.
type CookieSync struct {
	// DisabledIFrameBidders lists bidders for which iframe cookie syncs are disabled for
	// every account. Use "*" to disable iframe syncs for all bidders.
	DisabledIFrameBidders []string `mapstructure:"disabled_iframe_bidders" json:"disabled_iframe_bidders"`
}
