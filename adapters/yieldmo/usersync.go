package yieldmo

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewYieldmoSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("yieldmo", 173, temp, adapters.SyncTypeRedirect)
}
