package analyticsdeps

import (
	"net/http"

	"github.com/benbjohnson/clock"
)

// Deps to minimalny zestaw zależności wymaganych przez moduły analityczne.
// Użyjemy go w Kroku 3, gdy zrefaktoryzujemy buildery.
type Deps struct {
	HTTPClient *http.Client
	Clock      clock.Clock
}
