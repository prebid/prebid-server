package adyoulike

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdyoulikeSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adyoulike", urlTemplate, adapters.SyncTypeRedirect)
}
