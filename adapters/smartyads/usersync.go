package smartyads

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewSmartyAdsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("smartyads", 0, temp, adapters.SyncTypeRedirect)
}
