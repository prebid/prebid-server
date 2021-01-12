package acuityads

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAcuityAdsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("acuityads", 231, temp, adapters.SyncTypeRedirect)
}
