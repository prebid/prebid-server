package config_test

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("pbs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/config")

	viper.SetDefault("external_url", "http://localhost:8000")
	viper.SetDefault("port", 8000)
	viper.SetDefault("admin_port", 6060)
	viper.SetDefault("default_timeout_ms", 250)
	viper.SetDefault("datacache.type", "dummy")

	viper.SetDefault("adapters.pubmatic.endpoint", "http://openbid-useast.pubmatic.com/translator?")
	viper.SetDefault("adapters.rubicon.endpoint", "http://staged-by.rubiconproject.com/a/api/exchange.json")
	viper.SetDefault("adapters.rubicon.usersync_url", "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid")
	viper.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
}

func TestNewConfig(t *testing.T) {

	cfg, err := config.New()
	if err != nil {
		t.Error(err.Error())
	}

	if cfg.Port != 8000 {
		t.Error("Expected Port 8000")
	}

	if cfg.AdminPort != 6060 {
		t.Error("Expected Admin Port 6060")
	}

	if cfg.DefaultTimeout != uint64(250) {
		t.Error("Expected DefaultTimeout of 250ms")
	}

	if cfg.DataCache.Type != "dummy" {
		t.Error("Expected DataCache Type of 'dummy'")
	}

	if cfg.Adapters["pubmatic"].Endpoint != "http://openbid-useast.pubmatic.com/translator?" {
		t.Errorf("Expected Pubmatic Endpoint of http://openbid-useast.pubmatic.com/translator?")
	}

}
