package pubmatic

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewPubmaticSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("pubmatic", 76, temp, adapters.SyncTypeIframe)
}
