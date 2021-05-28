package between

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewBetweenSyncer returns "between" syncer
func NewBetweenSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("between", template, adapters.SyncTypeRedirect)
}
