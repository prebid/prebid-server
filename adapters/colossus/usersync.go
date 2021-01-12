package colossus

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

// NewColossusSyncer returns colossus syncer
func NewColossusSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("colossus", 0, temp, adapters.SyncTypeRedirect)
}
