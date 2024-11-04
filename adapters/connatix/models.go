package connatix

import (
	"net/url"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type adapter struct {
	uri url.URL
}

type impExtIncoming struct {
	Bidder openrtb_ext.ExtImpConnatix `json:"bidder"`
}

type impExt struct {
	Connatix impExtConnatix `json:"connatix"`
}

type impExtConnatix struct {
	PlacementId string `json:"placementId,omitempty"`
}

type bidExt struct {
	Cnx bidCnxExt `json:"connatix,omitempty"`
}

type bidCnxExt struct {
	MediaType string `json:"mediaType,omitempty"`
}
