package server

import (
	// import for side-effects
	"flag"
	"math/rand"
	_ "net/http/pprof"
	"time"

	_ "github.com/lib/pq"

	"github.com/golang/glog"
	"github.com/spf13/viper"

	"github.com/prebid/prebid-server/server"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse() // read glog settings from cmd line
}

func main() {
	viper.SetConfigName("pbs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/config")

	viper.SetDefault("external_url", "http://localhost:8000")
	viper.SetDefault("port", 8000)
	viper.SetDefault("admin_port", 6060)
	viper.SetDefault("default_timeout_ms", 250)
	viper.SetDefault("datacache.type", "dummy")

	// no metrics configured by default (metrics{host|database|username|password})

	viper.SetDefault("pubmatic_endpoint", "http://openbid-useast.pubmatic.com/translator?")
	viper.SetDefault("rubicon_endpoint", "http://staged-by.rubiconproject.com/a/api/exchange.json")
	viper.SetDefault("rubicon_usersync_url", "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid")
	viper.SetDefault("pulsepoint_endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	viper.ReadInConfig()

	s, err := server.NewServer(nil, nil)
	if err != nil {
		glog.Fatal(err)
	}
	s.Run(nil)
}
