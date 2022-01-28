package exchange

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/gdpr"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/stored_requests"
)

func BuildTestExchange(adapterMap map[openrtb_ext.BidderName]AdaptedBidder, categoriesFetcher stored_requests.CategoryFetcher, currencyConverter *currency.RateConverter) *exchange {
	return &exchange{
		adapterMap: adapterMap,
		me:         &metricsConfig.NilMetricsEngine{},
		cache:      &wellBehavedCache{},
		cacheTime:  time.Duration(0),
		gDPR:       gdpr.AlwaysAllow{},
		//currencyConverter: currency.NewRateConverter(&http.Client{}, "", time.Duration(0)),
		currencyConverter: currencyConverter,
		gdprDefaultValue:  gdpr.SignalYes,
		categoriesFetcher: categoriesFetcher,
		bidIDGenerator:    &mockBidIDGenerator{false, false},
	}
}

type wellBehavedCache struct{}

func (c *wellBehavedCache) GetExtCacheData() (scheme string, host string, path string) {
	return "https", "www.pbcserver.com", "/pbcache/endpoint"
}

func (c *wellBehavedCache) PutJson(ctx context.Context, values []pbc.Cacheable) ([]string, []error) {
	ids := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		ids[i] = strconv.Itoa(i)
	}
	return ids, nil
}

type mockBidIDGenerator struct {
	GenerateBidID bool `json:"generateBidID"`
	ReturnError   bool `json:"returnError"`
}

func (big *mockBidIDGenerator) Enabled() bool {
	return big.GenerateBidID
}

func (big *mockBidIDGenerator) New() (string, error) {

	if big.ReturnError {
		err := errors.New("Test error generating bid.ext.prebid.bidid")
		return "", err
	}
	return "mock_uuid", nil

}
