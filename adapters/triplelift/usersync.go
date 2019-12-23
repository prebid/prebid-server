package triplelift

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewTripleliftSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("triplelift", 28, temp, adapters.SyncTypeRedirect)
}
