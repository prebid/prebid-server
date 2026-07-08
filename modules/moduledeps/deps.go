package moduledeps

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prebid/prebid-server/v4/currency"
)

// ModuleDeps provides dependencies that custom modules may need for hooks execution.
// Additional dependencies can be added here if modules need something more.
type ModuleDeps struct {
	HTTPClient    *http.Client
	RateConvertor *currency.RateConverter
	Geoscope      map[string][]string
	// MetricsRegisterer lets a module register its own Prometheus collectors into the registry the
	// server scrapes at /metrics. May be nil (e.g. when Prometheus is disabled), in which case a
	// module must treat metrics as a no-op.
	MetricsRegisterer prometheus.Registerer
}
