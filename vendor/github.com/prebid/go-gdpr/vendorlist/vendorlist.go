package vendorlist

import (
	"github.com/prebid/go-gdpr/api"
)

// Copying from API for backwards compatibility

type VendorList interface {
	api.VendorList
}

type Vendor interface {
	api.Vendor
}
