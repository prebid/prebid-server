package router

import (
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/endpoints"
)

func Admin(revision string, rateConverter *currency.RateConverter, rateConverterFetchingInterval time.Duration) *http.ServeMux {
	// Add endpoints to the admin server
	// Making sure to add pprof routes
	mux := http.NewServeMux()
	// Register pprof handlers
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	// Register prebid-server defined admin handlers
	mux.HandleFunc("/currency/rates", endpoints.NewCurrencyRatesEndpoint(rateConverter, rateConverterFetchingInterval))
	mux.HandleFunc("/version", endpoints.NewVersionEndpoint(revision))
	return mux
}
