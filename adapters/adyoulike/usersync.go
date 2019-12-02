package adyoulike

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewadyoulikeSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adyoulike", 0, urlTemplate, adapters.SyncTypeRedirect)
}
