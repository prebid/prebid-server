package onetag

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("onetag", template, adapters.SyncTypeIframe)
}
