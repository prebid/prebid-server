package adtarget

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAdtargetSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adtarget", 0, temp, adapters.SyncTypeRedirect)
}
