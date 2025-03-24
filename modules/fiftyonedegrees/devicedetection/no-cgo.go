//go:build !cgo

package devicedetection

func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	panic("Do not enable the fiftyonedegrees module unless CGO is enabled")
}
