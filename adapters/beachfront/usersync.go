package beachfront

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewBeachfrontSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("beachfront", 0, temp, adapters.SyncTypeIframe)
}
