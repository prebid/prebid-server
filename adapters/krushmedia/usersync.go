package krushmedia

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewKrushmediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("krushmedia", 0, temp, adapters.SyncTypeRedirect)
}
