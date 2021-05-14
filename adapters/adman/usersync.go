package adman

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewAdmanSyncer returns adman syncer
func NewAdmanSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adman", temp, adapters.SyncTypeRedirect)
}
