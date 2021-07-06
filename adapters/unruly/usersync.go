package unruly

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewUnrulySyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("unruly", temp, adapters.SyncTypeIframe)
}
