package prebidServer

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/currencies"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
	"github.com/julienschmidt/httprouter"

	pbc "github.com/PubMatic-OpenWrap/prebid-server/prebid_cache_client"
	"github.com/PubMatic-OpenWrap/prebid-server/router"

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

const schemaDirectory = "/home/http/GO_SERVER/dmhbserver/static/"

/*
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
*/

func InitPrebidServer(configFile string) {
	rand.Seed(time.Now().UnixNano())
	v := viper.New()
	config.SetupViper(v, configFile)
	v.SetConfigFile(configFile)
	v.ReadInConfig()

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

	_, err := router.New(cfg, currencyConverter)
	if err != nil {
		return err
	}
	// Init prebid cache
	pbc.InitPrebidCache(cfg.CacheURL.GetBaseURL())
	pbc.InitPrebidCacheURL(cfg.ExternalURL)

	// Add cors support
	//corsRouter := router.SupportCORS(r)
	//server.Listen(cfg, router.NoCache{Handler: corsRouter}, router.Admin(revision, currencyConverter), r.MetricsEngine)
	//r.Shutdown()
	return nil
}

func OrtbAuction(w http.ResponseWriter, r *http.Request) error {
	return router.OrtbAuctionEndpointWrapper(w, r)
}

func Auction(w http.ResponseWriter, r *http.Request) {
	router.AuctionWrapper(w, r)

}

func GetUIDS(w http.ResponseWriter, r *http.Request) {
	router.GetUIDSWrapper(w, r)
}

func SetUIDS(w http.ResponseWriter, r *http.Request) {
	router.SetUIDSWrapper(w, r)
}

func CookieSync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	router.CookieSync(w, r)
}

func SyncerMap() map[openrtb_ext.BidderName]usersync.Usersyncer {
	return router.SyncerMap()
}
