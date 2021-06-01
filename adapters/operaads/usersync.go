package operaads

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewOperaadsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("operaads", temp, adapters.SyncTypeRedirect)
}
