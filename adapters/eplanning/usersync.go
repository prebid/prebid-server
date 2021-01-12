package eplanning

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewEPlanningSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("eplanning", 90, temp, adapters.SyncTypeIframe)
}
