package adtarget

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdtargetSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adtarget", temp, adapters.SyncTypeIframe)
}
