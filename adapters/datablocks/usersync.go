package datablocks

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

const datablocksGDPRVendorID = uint16(0)

func NewDatablocksSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("datablocks", datablocksGDPRVendorID, temp, adapters.SyncTypeRedirect)
}
