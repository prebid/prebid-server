package acuityads

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAcuityAdsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("acuityads", temp, adapters.SyncTypeRedirect)
}
