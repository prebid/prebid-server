package mediafuse

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewMediafuseSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("mediafuse", temp, adapters.SyncTypeIframe)
}
