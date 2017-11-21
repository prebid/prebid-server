package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

// Configuration
type Configuration struct {
	ExternalURL     string             `mapstructure:"external_url"`
	Host            string             `mapstructure:"host"`
	Port            int                `mapstructure:"port"`
	AdminPort       int                `mapstructure:"admin_port"`
	DefaultTimeout  uint64             `mapstructure:"default_timeout_ms"`
	CacheURL        Cache              `mapstructure:"cache"`
	RecaptchaSecret string             `mapstructure:"recaptcha_secret"`
	HostCookie      HostCookie         `mapstructure:"host_cookie"`
	Metrics         Metrics            `mapstructure:"metrics"`
	DataCache       DataCache          `mapstructure:"datacache"`
	Adapters        map[string]Adapter `mapstructure:"adapters"`
}

type HostCookie struct {
	Domain     string `mapstructure:"domain"`
	Family     string `mapstructure:"family"`
	CookieName string `mapstructure:"cookie_name"`
	OptOutURL  string `mapstructure:"opt_out_url"`
	OptInURL   string `mapstructure:"opt_in_url"`
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
	Host     string `mapstructure:"host"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
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

type Cache struct {
	Scheme string `mapstructure:"scheme"`
	Host   string `mapstructure:"host"`
	Query  string `mapstructure:"query"`
}

// New uses viper to get our server configurations
func New() (*Configuration, error) {
	var c Configuration
	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

//Allows for protocol relative URL if scheme is empty
func (cfg *Configuration) GetCacheBaseURL() string {
	cfg.CacheURL.Scheme = strings.ToLower(cfg.CacheURL.Scheme)
	if strings.Contains(cfg.CacheURL.Scheme, "https") {
		return fmt.Sprintf("https://%s", cfg.CacheURL.Host)
	}
	if strings.Contains(cfg.CacheURL.Scheme, "http") {
		return fmt.Sprintf("http://%s", cfg.CacheURL.Host)
	}
	return fmt.Sprintf("//%s", cfg.CacheURL.Host)
}

func (cfg *Configuration) GetCachedAssetURL(uuid string) string {
	return fmt.Sprintf("%s/cache?%s", cfg.GetCacheBaseURL(), strings.Replace(cfg.CacheURL.Query, "%PBS_CACHE_UUID%", uuid, 1))
}
