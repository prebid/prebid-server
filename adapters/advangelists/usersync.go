package advangelists

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAdvangelistsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("advangelists", 0, temp, adapters.SyncTypeIframe)
}
