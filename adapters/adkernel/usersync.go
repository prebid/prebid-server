package adkernel

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

const adkernelGDPRVendorID = uint16(14)

func NewAdkernelSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adkernel", 14, temp, adapters.SyncTypeRedirect)
}
