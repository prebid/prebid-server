package adocean

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdOceanSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adocean", temp, adapters.SyncTypeRedirect)
}
