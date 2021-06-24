package playwire

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewPlaywireSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("playwire", temp, adapters.SyncTypeRedirect)
}
