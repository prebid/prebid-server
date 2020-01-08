package adpone

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
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
