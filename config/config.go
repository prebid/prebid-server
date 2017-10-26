package config

import (
	"bytes"
	"github.com/spf13/viper"
	"strings"
	"fmt"
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
	Host  string `mapstructure:"host"`
	Query string `mapstructure:"query"`
}

// New uses viper to get our server configurations
func New() (*Configuration, error) {
	var c Configuration
	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (cfg *Configuration) GetCacheURL(uuid string) string {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s://%s/cache?%s", cfg.CacheURL.Scheme,cfg.CacheURL.Host, strings.Replace(cfg.CacheURL.Query, "%PBS_CACHE_UUID%", uuid, 1))
	return buffer.String()
}
