//go:build !cgo

package vastlint

import "errors"

func newValidator() (func(string) ([]finding, error), error) {
	return nil, errors.New("openadtech.vastlint requires cgo")
}
