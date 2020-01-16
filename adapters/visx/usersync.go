package visx

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewVisxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("visx", 0, temp, adapters.SyncTypeRedirect)
}
