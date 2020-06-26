package yieldlab

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewYieldlabSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("yieldlab", 70, temp, adapters.SyncTypeRedirect)
}
