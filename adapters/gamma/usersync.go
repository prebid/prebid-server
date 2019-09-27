package gamma

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewGammaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("gamma", 0, temp, adapters.SyncTypeIframe)
}
