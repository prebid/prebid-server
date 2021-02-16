package logan

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

// NewLoganSyncer retrun new syncer
func NewLoganSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("logan", 0, temp, adapters.SyncTypeRedirect)
}
