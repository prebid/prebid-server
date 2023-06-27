package config

type AllowActivities struct {
	SyncUser                 Activity `mapstructure:"syncUser" json:"syncUser"`
	FetchBids                Activity `mapstructure:"fetchBids" json:"fetchBids"`
	EnrichUserFPD            Activity `mapstructure:"enrichUfpd" json:"enrichUfpd"`
	ReportAnalytics          Activity `mapstructure:"reportAnalytics" json:"reportAnalytics"`
	TransmitUserFPD          Activity `mapstructure:"transmitUfpd" json:"transmitUfpd"`
	TransmitPreciseGeo       Activity `mapstructure:"transmitPreciseGeo" json:"transmitPreciseGeo"`
	TransmitUniqueRequestIds Activity `mapstructure:"transmitUniqueRequestIds" json:"transmitUniqueRequestIds"`
}

type Activity struct {
	Default *bool          `mapstructure:"default" json:"default"`
	Rules   []ActivityRule `mapstructure:"rules" json:"rules"`
	Allow   bool           `mapstructure:"allow" json:"allow"`
}

type ActivityRule struct {
	Condition ActivityCondition `mapstructure:"condition" json:"condition"`
}

type ActivityCondition struct {
	ComponentName []string `mapstructure:"componentName" json:"componentName"`
	ComponentType []string `mapstructure:"componentType" json:"componentType"`
}
