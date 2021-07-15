package openweb

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewOpenWebSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("openweb", temp, adapters.SyncTypeIframe)
}
