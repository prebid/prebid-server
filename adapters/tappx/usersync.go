package tappx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewTappxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("tappx", temp, adapters.SyncTypeIframe)
}
