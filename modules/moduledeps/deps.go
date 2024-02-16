package moduledeps

import (
	"net/http"

	"github.com/prebid/prebid-server/v2/currency"
	"github.com/prometheus/client_golang/prometheus"
)

// ModuleDeps provides dependencies that custom modules may need for hooks execution.
// Additional dependencies can be added here if modules need something more.
type ModuleDeps struct {
	HTTPClient         *http.Client
	RateConvertor      *currency.RateConverter
	PrometheusGatherer *prometheus.Registry
}
