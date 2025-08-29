package moduledeps

import (
	"net/http"

	"github.com/prebid/prebid-server/v3/currency"
)

// ModuleDeps provides dependencies that custom modules may need for hooks execution.
// Additional dependencies can be added here if modules need something more.
type ModuleDeps struct {
	HTTPClient    *http.Client
	RateConvertor *currency.RateConverter
}
