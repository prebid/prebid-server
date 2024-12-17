//go:build cgo

package modules

import (
	fiftyonedegreesDevicedetection "github.com/prebid/prebid-server/v3/modules/fiftyonedegrees/devicedetection"
)

func addFiftyonedegreesDevicedetection(b ModuleBuilders) {
	if b["fiftyonedegrees"] == nil {
		b["fiftyonedegrees"] = make(map[string]ModuleBuilderFn)
	}
	b["fiftyonedegrees"]["devicedetection"] = fiftyonedegreesDevicedetection.Builder
}
