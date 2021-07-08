package beachfront

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewBeachfrontSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		"beachfront",
		temp,
		adapters.SyncTypeIframe)
}
