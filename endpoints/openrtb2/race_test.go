package openrtb2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	metrics "github.com/rcrowley/go-metrics"
)

func TestRequestConcurrently(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(nobidServer))
	defer server.Close()

	cfg := &config.Configuration{}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	ex := exchange.NewExchange(server.Client(), &mockCache{}, cfg, theMetrics)
	paramsValidator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("failed to load params validators: %v", err)
	}

	endpoint, _ := NewEndpoint(ex, paramsValidator, empty_fetcher.EmptyFetcher(), &config.Configuration{MaxRequestSize: maxSize}, theMetrics)

	req := `
	{
		"id": "some-request-id",
		"site": {
			"page": "test.somepage.com"
		},
		"imp": [
			{
				"id": "my-imp-id",
				"banner": {
					"format": [
						{
							"w": 300,
							"h": 600
						}
					]
				},
				"pmp": {
					"deals": [
						{
							"id": "some-deal-id"
						}
					]
				},
				"ext": {
					"appnexus": {
						"placementId": 10433394
					},
					"rubicon": {
						"accountId": 1001,
						"siteId": 113932,
						"zoneId": 535510
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"targeting": {
					"pricegranularity": "low"
				},
				"cache": {
					"bids": {}
				}
			}
		}
	}
	`
	httpReq, err := http.NewRequest("POST", "pbs.com/openrtb2/auction", strings.NewReader(req))
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	r := httptest.NewRecorder()
	endpoint(r, httpReq, nil)
	if r.Code != http.StatusOK {
		t.Errorf("Got response status: %d. Expected %d", r.Code, http.StatusOK)
	}
}

func nobidServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(204)
}

type mockCache struct{}

func (c *mockCache) PutJson(ctx context.Context, values []json.RawMessage) (ids []string) {
	ids = make([]string, len(values))
	for i := 0; i < len(values); i++ {
		ids[i] = strconv.Itoa(i)
	}
	return
}
