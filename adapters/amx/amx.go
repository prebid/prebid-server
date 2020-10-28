package amx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const vastImpressionFormat = "<Impression><![CDATA[%s]]></Impression>"
const vastSearchPoint = "</Impression>"
const nbrHeaderName = "x-nbr"
const adapterVersion = "pbs1.0"

// AMXAdapter is the AMX bid adapter
type AMXAdapter struct {
	endpoint string
}

// NewAMXBidder creates an AMXAdapter
func NewAMXBidder(endpoint string) *AMXAdapter {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		glog.Fatalf("invalid endpoint provided to AMX: %s, error: %v", endpoint, err)
		return nil
	}

	qs, _ := url.ParseQuery(endpointURL.RawQuery)
	qs.Add("v", adapterVersion)
	endpointURL.RawQuery = qs.Encode()

	return &AMXAdapter{endpoint: endpointURL.String()}
}

type amxExt struct {
	Bidder openrtb_ext.ExtImpAMX `json:"bidder"`
}

func getTagID(imps []openrtb.Imp) (string, bool) {
	for _, imp := range imps {
		var params amxExt
		if err := json.Unmarshal(imp.Ext, &params); err == nil {
			if params.Bidder.TagID != "" {
				return params.Bidder.TagID, true
			}
		}
	}

	return "", false
}

func ensurePublisherWithID(pub *openrtb.Publisher, publisherID string) *openrtb.Publisher {
	if pub == nil {
		pub = &openrtb.Publisher{}
	}
	pub.ID = publisherID
	return pub
}

// MakeRequests creates AMX adapter requests
func (adapter *AMXAdapter) MakeRequests(request *openrtb.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	if publisherID, ok := getTagID(request.Imp); ok {
		if request.App != nil {
			request.App.Publisher = ensurePublisherWithID(request.App.Publisher, publisherID)
		}
		if request.Site != nil {
			request.Site.Publisher = ensurePublisherWithID(request.Site.Publisher, publisherID)
		}
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	reqBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    encoded,
		Headers: headers,
	}
	reqsBidder = append(reqsBidder, reqBidder)
	return
}

type amxBidExt struct {
	Himp       []string `json:"himp,omitempty"`
	StartDelay *int     `json:"startdelay,omitempty"`
}

// MakeBids will parse the bids from the AMX server
func (adapter *AMXAdapter) MakeBids(request *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if http.StatusNoContent == response.StatusCode {
		return nil, nil
	}

	if http.StatusBadRequest == response.StatusCode {
		internalNBR := response.Headers.Get(nbrHeaderName)
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid Request: 400. Error Code: %s", internalNBR),
		}}
	}

	if http.StatusOK != response.StatusCode {
		internalNBR := response.Headers.Get(nbrHeaderName)
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected response: %d. Error Code: %s", response.StatusCode, internalNBR),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidExt, bidExtErr := getBidExt(bid.Ext)
			if bidExtErr != nil {
				errs = append(errs, bidExtErr)
				continue
			}

			bidType := getMediaTypeForBid(bidExt)
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			if b.BidType == openrtb_ext.BidTypeVideo {
				b.Bid.AdM = interpolateImpressions(bid, bidExt)
				// remove the NURL so a client/player doesn't fire it twice
				b.Bid.NURL = ""
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs

}

func getBidExt(ext json.RawMessage) (amxBidExt, error) {
	if len(ext) == 0 {
		return amxBidExt{}, nil
	}

	var bidExt amxBidExt
	err := json.Unmarshal(ext, &bidExt)
	return bidExt, err
}

func getMediaTypeForBid(bidExt amxBidExt) openrtb_ext.BidType {
	if bidExt.StartDelay != nil {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

func pixelToImpression(pixel string) string {
	return fmt.Sprintf(vastImpressionFormat, pixel)
}

func interpolateImpressions(bid openrtb.Bid, ext amxBidExt) string {
	var buffer strings.Builder
	buffer.WriteString(pixelToImpression(bid.NURL))
	for _, impPixel := range ext.Himp {
		buffer.WriteString(pixelToImpression(impPixel))
	}

	results := strings.Replace(bid.AdM, vastSearchPoint, vastSearchPoint+buffer.String(), 1)
	return results
}
