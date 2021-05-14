package pubmatic

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewPubmaticSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("pubmatic", temp, adapters.SyncTypeIframe)
}
