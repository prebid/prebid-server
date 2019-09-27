package improvedigital

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewImprovedigitalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("improvedigital", 253, temp, adapters.SyncTypeRedirect)
}
