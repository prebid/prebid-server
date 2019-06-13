package sharethrough

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewSharethroughSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("sharethrough", 80, temp, adapters.SyncTypeRedirect)
}
