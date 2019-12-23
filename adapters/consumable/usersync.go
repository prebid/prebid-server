package consumable

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

var VENDOR_ID uint16 = 65535 // TODO: Insert consumable value when one is assigned

func NewConsumableSyncer(temp *template.Template) usersync.Usersyncer {

	return adapters.NewSyncer(
		"consumable",
		VENDOR_ID,
		temp,
		adapters.SyncTypeRedirect)
}
