package config

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/spf13/viper"
)

// Configuration
type Configuration struct {
	CookieDomain    string
	ExternalURL     string
	Host            string
	Port            int
	AdminPort       int
	DefaultTimeout  uint64
	DataCache       DataCache
	CacheURL        string
	RequireUUID2    bool
	RecaptchaSecret string
	Metrics         Metrics
	Adapters        []adapters.Configuration
}

// New uses viper to get our server configurations
func New() *Configuration {
	return &Configuration{
		CookieDomain:    viper.GetString("cookie_domain"),
		ExternalURL:     viper.GetString("external_url"),
		Host:            viper.GetString("host"),
		Port:            viper.GetInt("port"),
		AdminPort:       viper.GetInt("admin_port"),
		DefaultTimeout:  uint64(viper.GetInt("default_timeout_ms")),
		CacheURL:        viper.GetString("prebid_cache_url"),
		RequireUUID2:    viper.GetBool("require_uuid2"),
		RecaptchaSecret: viper.GetString("recaptcha_secret"),
		DataCache:       newdataCacheConfiguration(),
		Metrics:         newmetricsConfiguration(),
		//Adapters:        newadapters.Configurations(),
	}

}

type Metrics struct {
	Host     string
	Database string
	Username string
	Password string
}

func newmetricsConfiguration() Metrics {
	return Metrics{viper.GetString("metrics.host"), viper.GetString("metrics.database"), viper.GetString("metrics.username"), viper.GetString("metrics.password")}
}

type DataCache struct {
	Type       string
	Filename   string
	Database   string
	Host       string
	Username   string
	Password   string
	CacheSize  int
	TTLSeconds int
}

func newdataCacheConfiguration() DataCache {
	return DataCache{viper.GetString("datacache.type"), viper.GetString("datacache.filename"), viper.GetString("datacache.dbname"), viper.GetString("datacache.host"), viper.GetString("datacache.user"), viper.GetString("datacache.password"), viper.GetInt("datacache.cache_size"), viper.GetInt("datacache.ttl_seconds")}
}
