package liftoff

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewLiftoffSyncer ...
func NewLiftoffSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("liftoff", 667, temp, adapters.SyncTypeRedirect)
}
