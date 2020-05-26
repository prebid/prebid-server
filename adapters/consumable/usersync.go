package consumable

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

var VENDOR_ID uint16 = 591

func NewConsumableSyncer(temp *template.Template) usersync.Usersyncer {

	return adapters.NewSyncer(
		"consumable",
		VENDOR_ID,
		temp,
		adapters.SyncTypeRedirect)
}
