package amx

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strings"
)

const vastImpressionFormat = "<Impression><![CDATA[%s]></Impression>"
const vastSearchPoint = "</Impression>"

// AmxAdapter is the AMX bid adapter
type AmxAdapter struct {
	endpoint string
}

// NewAmxBidder creates an AmxAdapter
func NewAmxBidder(endpoint string) *AmxAdapter {
	return &AmxAdapter{endpoint: endpoint}
}

type amxExt struct {
	Bidder amxParams `json:"bidder"`
}

type amxParams struct {
	PublisherID string `json:"tagId,omitempty"`
}

func getPublisherID(imps []openrtb.Imp) *string {
	for _, imp := range imps {
		paramsSource := (*json.RawMessage)(&imp.Ext)
		var params amxExt
		if err := json.Unmarshal(*paramsSource, &params); err == nil {
			if params.Bidder.PublisherID != "" {
				return &params.Bidder.PublisherID
			}
		} else {
			fmt.Printf("unable to decode imp.ext.bidder: %v", err)
		}
	}

	return nil
}

func ensurePublisherWithID(pub *openrtb.Publisher, publisherID string) *openrtb.Publisher {
	if pub == nil {
		pub = &openrtb.Publisher{}
	}
	pub.ID = publisherID
	return pub
}

// MakeRequests creates AMX adapter requests
func (adapter *AmxAdapter) MakeRequests(request *openrtb.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	publisherID := getPublisherID(request.Imp)
	if publisherID != nil {
		if request.App != nil {
			request.App.Publisher = ensurePublisherWithID(request.App.Publisher, *publisherID)
		}
		if request.Site != nil {
			request.Site.Publisher = ensurePublisherWithID(request.Site.Publisher, *publisherID)
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
		Uri:     adapter.endpoint + "?v=pbs1.0",
		Body:    encoded,
		Headers: headers,
	}
	reqsBidder = append(reqsBidder, reqBidder)
	return
}

// MakeBids will parse the bids from the AMX server
func (adapter *AmxAdapter) MakeBids(request *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if http.StatusNoContent == response.StatusCode {
		return nil, nil
	}

	if http.StatusBadRequest == response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprint("Invalid request: 400"),
		}}
	}

	if http.StatusOK != response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected response: %d", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidType, err := getMediaTypeForImp(bid.ImpID, request.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				}
				if b.BidType == openrtb_ext.BidTypeVideo {
					b.Bid.AdM = interpolateImpressions(b.Bid)
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs

}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType, nil
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("No impression for: %s", impID),
	}
}

func pixelToImpression(pixel string) string {
	return fmt.Sprintf(vastImpressionFormat, pixel)
}

type amxBidResponseExt struct {
	Himp []string `json:"himp"`
}

func interpolateImpressions(bid *openrtb.Bid) string {
	var buffer strings.Builder
	buffer.WriteString(pixelToImpression(bid.NURL))

	ext := amxBidResponseExt{}
	if err := json.Unmarshal(bid.Ext, &ext); err == nil {
		for _, impPixel := range ext.Himp {
			buffer.WriteString(pixelToImpression(impPixel))
		}
	}
	results := strings.Replace(bid.AdM, vastSearchPoint, buffer.String(), 1)
	return results
}
