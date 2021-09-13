package colossus

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewColossusSyncer returns colossus syncer
func NewColossusSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("colossus", temp, adapters.SyncTypeRedirect)
}
