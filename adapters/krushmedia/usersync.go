package krushmedia

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewKrushmediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("krushmedia", temp, adapters.SyncTypeRedirect)
}
