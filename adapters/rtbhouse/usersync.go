package rtbhouse

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

const rtbHouseGDPRVendorID = uint16(16)
const rtbHouseFamilyName = "rtbhouse"

func NewRTBHouseSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		rtbHouseFamilyName,
		rtbHouseGDPRVendorID,
		urlTemplate,
		adapters.SyncTypeRedirect,
	)
}
