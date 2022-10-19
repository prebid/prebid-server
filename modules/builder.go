// Package modules, the content of the file generated automatically; DO NOT EDIT.
package modules

import (
	"github.com/prebid/prebid-server/modules/acme/foobar"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	return ModuleBuilders{
		"acme": {
			"foobar": foobar.Builder,
		},
	}
}
