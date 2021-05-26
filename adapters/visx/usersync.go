package visx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewVisxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("visx", temp, adapters.SyncTypeRedirect)
}
