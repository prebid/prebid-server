package smarthub

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSmartHubSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("smarthub", temp, adapters.SyncTypeRedirect)
}
