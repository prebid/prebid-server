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
	// MetricsRegisterer lets a module register its own Prometheus collectors into a registry the
	// server exports on the /metrics endpoint. May be nil, in which case a module must treat its
	// own metrics as a no-op.
	MetricsRegisterer prometheus.Registerer
}
