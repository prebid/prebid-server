package operaads

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

const operaAdsFamilyName = "operaads"

func NewOperaAdsSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		operaAdsFamilyName,
		urlTemplate,
		adapters.SyncTypeRedirect,
	)
}
