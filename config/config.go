package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"bytes"
	"strconv"
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
	ORTB2Config     OpenRTB2Config     `mapstructure:"ortb2_config"`
	Adapters        map[string]Adapter `mapstructure:"adapters"`
}

func (cfg *Configuration) validate() error {
	return cfg.ORTB2Config.validate()
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

// OpenRTB2Config configures the backend used to store Account and Request config data.
type OpenRTB2Config struct {
	// Files should be true if OpenRTB configs should be loaded from the filesystem
	Files bool `mapstructure:"filesystem"`
	// Postgres should be non-nil if OpenRTB configs should be loaded from a Postgres database
	Postgres *PostgresConfig `mapstructure:"postgres"`
}

func (cfg *OpenRTB2Config) validate() error {
	if cfg.Files && cfg.Postgres != nil {
		return errors.New("Only one of [filesystem, postgres] can be used at the same time.")
	}

	return nil
}

type PostgresConfig struct {
	Database string `mapstructure:"dbname"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"user"`
	Password string `mapstructure:"password"`

	// Query is the Postgres Query which can be used to fetch configs from the database.
	//
	// In the simplest case, Prebid Server expects this to be something like:
	//   SELECT id, config FROM table WHERE id in %ID_LIST%
	//
	// The MakeQuery function will transform this query into:
	//   SELECT id, config FROM table WHERE id in ($1, $2, $3, ...)
	//
	// ... where the number of "$x" args depends on how many configs need to be fetched in one request.
	Query string `mapstructure:"query"`
}

// MakeQuery gets a config-fetching query which can be used to fetch numConfigs configs at once.
func (cfg *PostgresConfig) MakeQuery(numConfigs int) (string, error) {
	if numConfigs < 1 {
		return "", fmt.Errorf("can't generate query to fetch %d configs", numConfigs)
	}
	final := bytes.NewBuffer(make([]byte, 0, 2 + 4 * numConfigs))
	final.WriteString("(")
	for i := 1; i < numConfigs; i++ {
		final.WriteString("$")
		final.WriteString(strconv.Itoa(i))
		final.WriteString(", ")
	}
	final.WriteString("$")
	final.WriteString(strconv.Itoa(numConfigs))
	final.WriteString(")")
	return strings.Replace(cfg.Query, "%ID_LIST%", final.String(), 1), nil
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
	return &c, c.validate()
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
