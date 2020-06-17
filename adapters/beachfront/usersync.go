package beachfront

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

var VENDOR_ID uint16 = 335

func NewBeachfrontSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		"beachfront",
		VENDOR_ID,
		temp,
		adapters.SyncTypeIframe)
}
