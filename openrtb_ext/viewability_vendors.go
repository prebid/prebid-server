package openrtb_ext

import "fmt"

var viewabilityVendorMap = map[string]string{
	"moat":         "moat.com",
	"adform":       "adform.com",
	"activeview":   "doubleclickbygoogle.com",
	"doubleverify": "doubleverify.com",
	"comscore":     "comscore.com",
	"integralads":  "integralads.com",
	"sizemek":      "sizemek.com",
	"whiteops":     "whiteops.com",
}

func GetVendorUrl(vendor string) (string, error) {
	if vendorUrl, ok := viewabilityVendorMap[vendor]; !ok {
		return "", fmt.Errorf("Vendor unknown: %v", vendor)
	} else {
		return vendorUrl, nil
	}
}
