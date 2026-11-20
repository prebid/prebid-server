package moduledeps

import (
	"net/http"

	"github.com/prebid/prebid-server/v4/currency"
	"github.com/prometheus/client_golang/prometheus"
)

// ModuleDeps provides dependencies that custom modules may need for hooks execution.
// Additional dependencies can be added here if modules need something more.
type ModuleDeps struct {
	HTTPClient    *http.Client
	RateConvertor *currency.RateConverter
	Geoscope      map[string][]string

	// MetricsRegisterer allows modules to expose collectors through Prebid Server's
	// existing Prometheus endpoint. It is nil when Prometheus is disabled.
	MetricsRegisterer prometheus.Registerer
}
