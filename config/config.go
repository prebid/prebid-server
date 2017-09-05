package config

import (
	"github.com/spf13/viper"
	"time"
)

// Configuration
type Configuration struct {
	CookieDomain    string             `mapstructure:"cookie_domain"`
	ExternalURL     string             `mapstructure:"external_url"`
	Host            string             `mapstructure:"host"`
	Port            int                `mapstructure:"port"`
	AdminPort       int                `mapstructure:"admin_port"`
	DefaultTimeout  uint64             `mapstructure:"default_timeout_ms"`
	CacheURL        string             `mapstructure:"prebid_cache_url"`
	RequireUUID2    bool               `mapstructure:"require_uuid2"`
	RecaptchaSecret string             `mapstructure:"recaptcha_secret"`
	Metrics         Metrics            `mapstructure:"metrics"`
	DataCache       DataCache          `mapstructure:"datacache"`
	Adapters        map[string]Adapter `mapstructure:"adapters"`
}

type Adapter struct {
	Endpoint    string `mapstructure:"endpoint"` // Required
	UserSyncURL string `mapstructure:"usersync_url"`
	PlatformID  string `mapstructure:"platform_id"` // needed for Facebook
	XAPI        struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Tracker  string `mapstructure:"tracker"`
	} `mapstructure:"xapi"` // needed for Rubicon
}

type Metrics struct {
	Type     string        `mapstructure:"type"`
	Host     string        `mapstructure:"host"`
	Database string        `mapstructure:"database"`
	Username string        `mapstructure:"username"`
	Password string        `mapstructure:"password"`
	Interval time.Duration `mapstructure:"interval"`
	Prefix   string        `mapstructure:"prefix"`
}

type DataCache struct {
	Type       string `mapstructure:"type"`
	Filename   string `mapstructure:"filename"`
	Database   string `mapstructure:"dbname"`
	Host       string `mapstructure:"host"`
	Username   string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	CacheSize  int    `mapstructure:"cache_size"`
	TTLSeconds int    `mapstructure:"ttl_seconds"`
}

// New uses viper to get our server configurations
func New() (*Configuration, error) {
	var c Configuration
	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}
