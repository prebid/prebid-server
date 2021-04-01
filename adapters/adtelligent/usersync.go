package adtelligent

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdtelligentSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adtelligent", temp, adapters.SyncTypeIframe)
}
