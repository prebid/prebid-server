package inmobi

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewInmobiSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("inmobi", template, adapters.SyncTypeRedirect)
}
