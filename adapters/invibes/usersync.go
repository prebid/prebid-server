package invibes

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewInvibesSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		"invibes",
		urlTemplate,
		adapters.SyncTypeIframe,
	)
}
