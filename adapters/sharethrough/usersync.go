package sharethrough

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSharethroughSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("sharethrough", temp, adapters.SyncTypeRedirect)
}
