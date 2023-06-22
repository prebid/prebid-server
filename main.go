package main

import (
	"flag"
	"math/rand"
	"net/http"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/router"
	"github.com/prebid/prebid-server/server"
	"github.com/prebid/prebid-server/util/task"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	flag.Parse() // required for glog flags and testing package flags

	bidderInfoPath, err := filepath.Abs(infoDirectory)
	if err != nil {
		glog.Exitf("Unable to build configuration directory path: %v", err)
	}

	bidderInfos, err := config.LoadBidderInfoFromDisk(bidderInfoPath)
	if err != nil {
		glog.Exitf("Unable to load bidder configurations: %v", err)
	}
	cfg, err := loadConfig(bidderInfos)
	if err != nil {
		glog.Exitf("Configuration could not be loaded or did not pass validation: %v", err)
	}

	setGarbageCollectionConfig(cfg)

	err = serve(cfg)
	if err != nil {
		glog.Exitf("prebid-server failed: %v", err)
	}
}

const configFileName = "pbs"
const infoDirectory = "./static/bidder-info"

func loadConfig(bidderInfos config.BidderInfos) (*config.Configuration, error) {
	v := viper.New()
	config.SetupViper(v, configFileName, bidderInfos)
	return config.New(v, bidderInfos, openrtb_ext.NormalizeBidderName)
}

func setGarbageCollectionConfig(cfg *config.Configuration) {
	// Create a soft memory limit on the total amount of memory that PBS uses to tune the behavior
	// of the Go garbage collector. In summary, `cfg.GarbageCollectorThreshold` serves as a fixed cost
	// of memory that is going to be held garbage before a garbage collection cycle is triggered.
	// This amount of virtual memory won’t translate into physical memory allocation unless we attempt
	// to read or write to the slice below, which PBS will not do.
	//
	// Note that since Go 1.19, the GOMEMLIMIT environment variable can be used to achieve the same, when
	// used in conjunction with GOGC.
	// PBS supports that with the `go_runtime.soft_memory_limit` and `go_runtime.gc_percent` configuration
	// options and this code will be removed once the deprecated `garbage_collector_threshold` field is removed.
	if cfg.GarbageCollectorThreshold > 0 {
		garbageCollectionThreshold := make([]byte, cfg.GarbageCollectorThreshold)
		defer runtime.KeepAlive(garbageCollectionThreshold)
	}

	// Note that setting the memory limit to a value equal to 0 or less than what's needed by the Go
	// runtime may cause the garbage collector to run continuously.
	if cfg.GoRuntime.SoftMemoryLimit >= 0 {
		debug.SetMemoryLimit(int64(cfg.GoRuntime.SoftMemoryLimit))
	}
	debug.SetGCPercent(cfg.GoRuntime.GarbageCollectorTriggerPercent)
}

func serve(cfg *config.Configuration) error {
	fetchingInterval := time.Duration(cfg.CurrencyConverter.FetchIntervalSeconds) * time.Second
	staleRatesThreshold := time.Duration(cfg.CurrencyConverter.StaleRatesSeconds) * time.Second
	currencyConverter := currency.NewRateConverter(&http.Client{}, cfg.CurrencyConverter.FetchURL, staleRatesThreshold)

	currencyConverterTickerTask := task.NewTickerTask(fetchingInterval, currencyConverter)
	currencyConverterTickerTask.Start()

	r, err := router.New(cfg, currencyConverter)
	if err != nil {
		return err
	}

	corsRouter := router.SupportCORS(r)
	server.Listen(cfg, router.NoCache{Handler: corsRouter}, router.Admin(currencyConverter, fetchingInterval), r.MetricsEngine)

	r.Shutdown()
	return nil
}
