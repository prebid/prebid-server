package adpone

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

const adponeGDPRVendorID = uint16(16)
const adponeFamilyName = "adpone"

func NewadponeSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		adponeFamilyName,
		adponeGDPRVendorID,
		urlTemplate,
		adapters.SyncTypeRedirect,
	)
}
