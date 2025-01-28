package connatix

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type adapter struct {
	endpoint string
}

type impExtIncoming struct {
	Bidder openrtb_ext.ExtImpConnatix `json:"bidder"`
}

type impExt struct {
	Connatix impExtConnatix `json:"connatix"`
}

type impExtConnatix struct {
	PlacementId           string  `json:"placementId,omitempty"`
	ViewabilityPercentage float64 `json:"viewabilityPercentage,omitempty"`
}

type bidExt struct {
	Cnx bidCnxExt `json:"connatix,omitempty"`
}

type bidCnxExt struct {
	MediaType string `json:"mediaType,omitempty"`
}
