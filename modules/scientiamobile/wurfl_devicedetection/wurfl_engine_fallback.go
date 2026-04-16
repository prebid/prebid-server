//go:build !wurfl

package wurfl_devicedetection

import "errors"

const wurflBuildTagMissingError = "wurfl module requires the wurfl build tag; build with: go build -tags wurfl"

func newWurflEngine(_ config) (wurflDeviceDetection, error) {
	return nil, errors.New(wurflBuildTagMissingError)
}
