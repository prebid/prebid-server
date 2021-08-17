package adf

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdfSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adf", temp, adapters.SyncTypeRedirect)
}
