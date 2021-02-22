package verizonmedia

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewVerizonMediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("verizonmedia", temp, adapters.SyncTypeRedirect)
}
