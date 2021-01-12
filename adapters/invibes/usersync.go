package invibes

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewInvibesSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		"invibes",
		436,
		urlTemplate,
		adapters.SyncTypeIframe,
	)
}
