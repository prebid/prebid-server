package main

import (
	"flag"
	"math/rand"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currencies"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/router"
	"github.com/prebid/prebid-server/server"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

// Rev holds binary revision string
// Set manually at build time using:
//    go build -ldflags "-X main.Rev=`git rev-parse --short HEAD`"
// Populated automatically at build / release time via .travis.yml
//   `gox -os="linux" -arch="386" -output="{{.Dir}}_{{.OS}}_{{.Arch}}" -ldflags "-X main.Rev=`git rev-parse --short HEAD`" -verbose ./...;`
// See issue #559
var Rev string

func init() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse() // read glog settings from cmd line
}

func main() {
	v := viper.New()
	config.SetupViper(v, "pbs")
	cfg, err := config.New(v)
	if err != nil {
		glog.Fatalf("Configuration could not be loaded or did not pass validation: %v", err)
	}
	if err := serve(Rev, cfg); err != nil {
		glog.Errorf("prebid-server failed: %v", err)
	}
}

func serve(revision string, cfg *config.Configuration) error {
	currencyConverter := currencies.NewRateConverter(&http.Client{}, cfg.CurrencyConverter.FetchURL, time.Duration(cfg.CurrencyConverter.FetchIntervalSeconds)*time.Second)
	r, err := router.New(cfg, currencyConverter)
	if err != nil {
		return err
	}
	// Init prebid cache
	pbc.InitPrebidCache(cfg.CacheURL.GetBaseURL())
	// Add cors support
	corsRouter := router.SupportCORS(r)
	server.Listen(cfg, router.NoCache{Handler: corsRouter}, router.Admin(revision, currencyConverter), r.MetricsEngine)
	r.Shutdown()
	return nil
}
