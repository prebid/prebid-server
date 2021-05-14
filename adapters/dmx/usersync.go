package dmx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewDmxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("dmx", temp, adapters.SyncTypeRedirect)
}
