package analyticsdeps

import (
	"net/http"

	"github.com/benbjohnson/clock"
)

type Deps struct {
	HTTPClient *http.Client
	Clock      clock.Clock
}
