package mgid

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewMgidSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("mgid", 358, temp, adapters.SyncTypeRedirect)
}
