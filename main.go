package prebidServer

import (
	"net/http"
	"time"

	"github.com/prebid/prebid-server/usersync"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/router"
	"github.com/prebid/prebid-server/util/task"

	"github.com/spf13/viper"
)

const schemaDirectory = "/home/http/GO_SERVER/dmhbserver/static/"

/*
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

	err = serve(cfg)
	if err != nil {
		glog.Exitf("prebid-server failed: %v", err)
	}
}
*/

const configFileName = "pbs"

func loadConfig(configFileName string) (*config.Configuration, error) {
	v := viper.New()
	config.SetupViper(v, configFileName)
	v.SetConfigFile(configFileName)
	v.ReadInConfig()
	return config.New(v)
}

func serve(cfg *config.Configuration) error {
	fetchingInterval := time.Duration(cfg.CurrencyConverter.FetchIntervalSeconds) * time.Second
	staleRatesThreshold := time.Duration(cfg.CurrencyConverter.StaleRatesSeconds) * time.Second
	currencyConverter := currency.NewRateConverter(&http.Client{}, cfg.CurrencyConverter.FetchURL, staleRatesThreshold)

	currencyConverterTickerTask := task.NewTickerTask(fetchingInterval, currencyConverter)
	currencyConverterTickerTask.Start()

	_, err := router.New(cfg, currencyConverter)
	if err != nil {
		return err
	}

	// corsRouter := router.SupportCORS(r)
	// server.Listen(cfg, router.NoCache{Handler: corsRouter}, router.Admin(currencyConverter, fetchingInterval), r.MetricsEngine)

	//r.Shutdown()

	// pbc.InitPrebidCache(cfg.CacheURL.GetBaseURL())
	// pbc.InitPrebidCacheURL(cfg.ExternalURL)

	return nil
}

func OrtbAuction(w http.ResponseWriter, r *http.Request) error {
	return router.OrtbAuctionEndpointWrapper(w, r)
}

var VideoAuction = func(w http.ResponseWriter, r *http.Request) error {
	return router.VideoAuctionEndpointWrapper(w, r)
}

func Auction(w http.ResponseWriter, r *http.Request) {
	// router.AuctionWrapper(w, r)
}

func GetUIDS(w http.ResponseWriter, r *http.Request) {
	router.GetUIDSWrapper(w, r)
}

func SetUIDS(w http.ResponseWriter, r *http.Request) {
	router.SetUIDSWrapper(w, r)
}

func CookieSync(w http.ResponseWriter, r *http.Request) {
	router.CookieSync(w, r)
}

func SyncerMap() map[string]usersync.Syncer {
	return router.SyncerMap()
}
