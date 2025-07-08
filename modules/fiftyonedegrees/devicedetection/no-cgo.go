//go:build !cgo

package devicedetection

import (
	"encoding/json"
	"errors"

	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

const errMsg = "fiftyonedegrees should not be enabled unless CGO is enabled"

func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	return nil, errors.New(errMsg)
}
