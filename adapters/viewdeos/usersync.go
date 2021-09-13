package viewdeos

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewViewdeosSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("viewdeos", temp, adapters.SyncTypeIframe)
}
