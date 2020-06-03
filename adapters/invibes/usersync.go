package invibes

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

const invibesGDPRVendorID = uint16(16)
const invibesFamilyName = "invibes"

func NewInvibesSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		invibesFamilyName,
		invibesGDPRVendorID,
		urlTemplate,
		adapters.SyncTypeRedirect,
	)
}
