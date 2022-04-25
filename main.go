package main

import (
	"flag"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/router"
	"github.com/prebid/prebid-server/server"
	"github.com/prebid/prebid-server/util/task"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	flag.Parse() // required for glog flags and testing package flags

	cfg, err := loadConfig()
	if err != nil {
		glog.Exitf("Configuration could not be loaded or did not pass validation: %v", err)
	}

	// Create a soft memory limit on the total amount of memory that PBS uses to tune the behavior
	// of the Go garbage collector. In summary, `cfg.GarbageCollectorThreshold` serves as a fixed cost
	// of memory that is going to be held garbage before a garbage collection cycle is triggered.
	// This amount of virtual memory won’t translate into physical memory allocation unless we attempt
	// to read or write to the slice below, which PBS will not do.
	garbageCollectionThreshold := make([]byte, cfg.GarbageCollectorThreshold)
	defer runtime.KeepAlive(garbageCollectionThreshold)

	// write PID to file for deploy management
	if cfg.DeployPIDEnabled {
		pid, err := handleDeployPID(cfg.DeployPIDPath, cfg.DeployPIDMode)
		if err != nil {
			glog.Fatalf("error writing pid[%d]: %s", pid, err)
		}
	}

	err = serve(cfg)
	if err != nil {
		glog.Exitf("prebid-server failed: %v", err)
	}
}

const configFileName = "pbs"

func loadConfig() (*config.Configuration, error) {
	v := viper.New()
	config.SetupViper(v, configFileName)
	return config.New(v)
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

	// setup tapjoy opentelemtry
	doneCB, err := initProvider(cfg.Monitoring.OpenTelemetry)
	if err != nil {
		return err
	}

	corsRouter := router.SupportCORS(r)
	// wrap corsRouter in otel handler
	otelHandler := otelhttp.NewHandler(corsRouter, "prebid/openrtb",
		// only wrap for "/openrtb2/auction" endpoint
		otelhttp.WithFilter(func(r *http.Request) bool {
			if r.RequestURI == "/openrtb2/auction" {
				return true
			}
			return false
		}),
	)
	server.Listen(cfg, router.NoCache{Handler: otelHandler}, router.Admin(currencyConverter, fetchingInterval), r.MetricsEngine)

	doneCB()
	r.Shutdown()
	return nil
}
