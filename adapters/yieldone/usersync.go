package yieldone

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewYieldoneSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("yieldone", 0, temp, adapters.SyncTypeRedirect)
}
