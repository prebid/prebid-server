package router

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/endpoints"
	"github.com/prebid/prebid-server/endpoints/openrtb2"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/uuidutil"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	g_syncers           map[string]usersync.Syncer
	g_cfg               *config.Configuration
	g_ex                *exchange.Exchange
	g_accounts          *stored_requests.AccountFetcher
	g_paramsValidator   *openrtb_ext.BidderParamValidator
	g_storedReqFetcher  *stored_requests.Fetcher
	g_storedRespFetcher *stored_requests.Fetcher
	g_gdprPerms         *gdpr.Permissions
	g_metrics           metrics.MetricsEngine
	g_analytics         *analytics.PBSAnalyticsModule
	g_disabledBidders   map[string]string
	g_categoriesFetcher *stored_requests.CategoryFetcher
	g_videoFetcher      *stored_requests.Fetcher
	g_activeBidders     map[string]openrtb_ext.BidderName
	g_defReqJSON        []byte
	g_cacheClient       *pbc.Client
	g_transport         *http.Transport
)

func getTransport(cfg *config.Configuration, certPool *x509.CertPool) *http.Transport {
	transport := &http.Transport{
		MaxConnsPerHost: cfg.Client.MaxConnsPerHost,
		IdleConnTimeout: time.Duration(cfg.Client.IdleConnTimeout) * time.Second,
		TLSClientConfig: &tls.Config{RootCAs: certPool},
	}

	if cfg.Client.DialTimeout > 0 {
		transport.Dial = (&net.Dialer{
			Timeout:   time.Duration(cfg.Client.DialTimeout) * time.Millisecond,
			KeepAlive: time.Duration(cfg.Client.DialKeepAlive) * time.Second,
		}).Dial
	}

	if cfg.Client.TLSHandshakeTimeout > 0 {
		transport.TLSHandshakeTimeout = time.Duration(cfg.Client.TLSHandshakeTimeout) * time.Second
	}

	if cfg.Client.ResponseHeaderTimeout > 0 {
		transport.ResponseHeaderTimeout = time.Duration(cfg.Client.ResponseHeaderTimeout) * time.Second
	}

	if cfg.Client.MaxIdleConns > 0 {
		transport.MaxIdleConns = cfg.Client.MaxIdleConns
	}

	if cfg.Client.MaxIdleConnsPerHost > 0 {
		transport.MaxIdleConnsPerHost = cfg.Client.MaxIdleConnsPerHost
	}

	return transport
}

func GetCacheClient() *pbc.Client {
	return g_cacheClient
}

func GetPrebidCacheURL() string {
	return g_cfg.ExternalURL
}

//OrtbAuctionEndpointWrapper Openwrap wrapper method for calling /openrtb2/auction endpoint
func OrtbAuctionEndpointWrapper(w http.ResponseWriter, r *http.Request) error {
	ortbAuctionEndpoint, err := openrtb2.NewEndpoint(uuidutil.UUIDRandomGenerator{}, *g_ex, *g_paramsValidator, *g_storedReqFetcher, *g_accounts, g_cfg, g_metrics, *g_analytics, g_disabledBidders, g_defReqJSON, g_activeBidders, *g_storedRespFetcher)
	if err != nil {
		return err
	}
	ortbAuctionEndpoint(w, r, nil)
	return nil
}

//VideoAuctionEndpointWrapper Openwrap wrapper method for calling /openrtb2/video endpoint
func VideoAuctionEndpointWrapper(w http.ResponseWriter, r *http.Request) error {
	videoAuctionEndpoint, err := openrtb2.NewCTVEndpoint(*g_ex, *g_paramsValidator, *g_storedReqFetcher, *g_videoFetcher, *g_accounts, g_cfg, g_metrics, *g_analytics, g_disabledBidders, g_defReqJSON, g_activeBidders)
	if err != nil {
		return err
	}
	videoAuctionEndpoint(w, r, nil)
	return nil
}

//GetUIDSWrapper Openwrap wrapper method for calling /getuids endpoint
func GetUIDSWrapper(w http.ResponseWriter, r *http.Request) {
	getUID := endpoints.NewGetUIDsEndpoint(g_cfg.HostCookie)
	getUID(w, r, nil)
}

//SetUIDSWrapper Openwrap wrapper method for calling /setuid endpoint
func SetUIDSWrapper(w http.ResponseWriter, r *http.Request) {
	setUID := endpoints.NewSetUIDEndpoint(g_cfg.HostCookie, g_syncers, *g_gdprPerms, *g_analytics, g_metrics)
	setUID(w, r, nil)
}

//CookieSync Openwrap wrapper method for calling /cookie_sync endpoint
func CookieSync(w http.ResponseWriter, r *http.Request) {
	cookiesync := endpoints.NewCookieSyncEndpoint(g_syncers, g_cfg, *g_gdprPerms, g_metrics, *g_analytics, g_activeBidders)
	cookiesync.Handle(w, r, nil)
}

//SyncerMap Returns map of bidder and its usersync info
func SyncerMap() map[string]usersync.Syncer {
	return g_syncers
}

func GetPrometheusGatherer() *prometheus.Registry {
	mEngine, ok := g_metrics.(*metricsConf.DetailedMetricsEngine)
	if !ok || mEngine == nil || mEngine.PrometheusMetrics == nil {
		return nil
	}

	return mEngine.PrometheusMetrics.Gatherer
}
