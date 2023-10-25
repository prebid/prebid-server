package moduledeps

import "net/http"

// ModuleDeps provides dependencies that custom modules may need for hooks execution.
// Additional dependencies can be added here if modules need something more.
type ModuleDeps struct {
	HTTPClient *http.Client
}
