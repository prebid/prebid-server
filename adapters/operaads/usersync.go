package operaads

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewOperaadsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("operaads", temp, adapters.SyncTypeRedirect)
}
