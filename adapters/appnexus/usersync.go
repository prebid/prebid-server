package appnexus

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAppnexusSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adnxs", temp, adapters.SyncTypeRedirect)
}
