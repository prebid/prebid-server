package smartadserver

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSmartadserverSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("smartadserver", temp, adapters.SyncTypeRedirect)
}
