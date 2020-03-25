package smartrtb

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSmartRTBSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("smartrtb", 0, temp, adapters.SyncTypeRedirect)
}
