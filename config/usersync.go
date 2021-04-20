package config

type UserSyncCooperative struct {
	Enabled        bool       `mapstructure:"enabled" json:"enabled,omitempty"`
	PriorityGroups [][]string `mapstructure:"priorityGroups" json:"priorityGroups,omitempty"`
}
