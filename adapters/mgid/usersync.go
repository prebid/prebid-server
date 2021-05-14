package mgid

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewMgidSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("mgid", temp, adapters.SyncTypeRedirect)
}
