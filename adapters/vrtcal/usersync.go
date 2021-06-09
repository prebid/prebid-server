package vrtcal

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewVrtcalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("vrtcal", temp, adapters.SyncTypeRedirect)
}
