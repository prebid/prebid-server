package smartyads

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSmartyAdsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("smartyads", 0, temp, adapters.SyncTypeRedirect)
}
