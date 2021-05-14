package openx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewOpenxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("openx", temp, adapters.SyncTypeRedirect)
}
