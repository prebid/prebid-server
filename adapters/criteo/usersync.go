package criteo

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewCriteoSyncer user syncer for Criteo
// Criteo doesn't need user synchronization yet.
func NewCriteoSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("criteo", temp, adapters.SyncTypeRedirect)
}
