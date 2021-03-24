package zemanta

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewZemantaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("zemanta", temp, adapters.SyncTypeRedirect)
}
