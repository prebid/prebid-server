package sonobi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// SonobiAdapter - Sonobi SonobiAdapter definition
type SonobiAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name returns the name fo cookie stuff
func (a *SonobiAdapter) Name() string {
	return "sonobi"
}

// NewSonobiAdapter create a new SovrnSonobiAdapter instance
func NewSonobiAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *SonobiAdapter {
	return NewSonobiBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

// NewSonobiBidder Initializes the Bidder
func NewSonobiBidder(client *http.Client, endpoint string) *SonobiAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &SonobiAdapter{
		http: a,
		URI:  endpoint,
	}
}

type sonobiParams struct {
	TagID string `json:"TagID"`
}

// MakeRequests
func (a *SonobiAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	var errs []error
	var sonobiExt openrtb_ext.ExtImpSoonobi
	var bannerImps []openrtb.Imp
	var videoImps []openrtb.Imp
	var err error

	for _, imp := range request.Imp {
		// Sonobi doesn't allow multi-type imp. Banner takes priority over video.
		if imp.Banner != nil {
			bannerImps = append(bannerImps, imp)
		} else if imp.Video != nil {
			videoImps = append(videoImps, imp)
		} else {
			err := fmt.Errorf("Sonobi only supports banner and video imps. Ignoring imp id=%s", imp.ID)
			errs = append(errs, err)
		}
	}

	var adapterRequests []*adapters.RequestData
	// Make a copy as we don't want to change the original request
	reqCopy := *request
	reqCopy.Imp = bannerImps
	reqCopy.Imp = append(reqCopy.Imp, videoImps...)

	for i := range reqCopy.Imp {
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(reqCopy.Imp[i].Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		if err = json.Unmarshal(bidderExt.Bidder, &sonobiExt); err != nil {

			errs = append(errs, err)
			continue
		}
		reqCopy.Imp[i].TagID = sonobiExt.TagID
	}

	adapterReq, errors := a.makeRequest(&reqCopy)
	if adapterReq != nil {
		adapterRequests = append(adapterRequests, adapterReq)
	}
	errs = append(errs, errors...)

	return adapterRequests, errs

}

// makeRequest helper method to crete the http request data
func (a *SonobiAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// MakeBids makes the bids
func (a *SonobiAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) ([]*adapters.TypedBid, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bids := make([]*adapters.TypedBid, 0, 5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bids = append(bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bids, nil
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}
