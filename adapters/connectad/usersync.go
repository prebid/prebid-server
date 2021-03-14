package connectad

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewConnectAdSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("connectad", temp, adapters.SyncTypeIframe)
}
