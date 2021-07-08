package gumgum

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewGumGumSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("gumgum", temp, adapters.SyncTypeIframe)
}
