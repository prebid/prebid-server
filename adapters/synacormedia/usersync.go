package synacormedia

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSynacorMediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("synacormedia", temp, adapters.SyncTypeIframe)
}
