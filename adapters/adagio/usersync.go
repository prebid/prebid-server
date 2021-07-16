package adagio

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdagioSyncer(tmpl *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adagio", tmpl, adapters.SyncTypeRedirect)
}
