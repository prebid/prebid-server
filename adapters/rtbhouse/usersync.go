package rtbhouse

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

const rtbHouseFamilyName = "rtbhouse"

func NewRTBHouseSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		rtbHouseFamilyName,
		urlTemplate,
		adapters.SyncTypeRedirect,
	)
}
