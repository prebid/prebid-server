package adman

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

// NewAdmanSyncer returns adman syncer
func NewAdmanSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adman", 149, temp, adapters.SyncTypeRedirect)
}
