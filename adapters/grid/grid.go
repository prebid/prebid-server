package grid

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type GridAdapter struct {
	endpoint string
}

type ExtImpDataAdServer struct {
	Name   string `json:"name"`
	AdSlot string `json:"adslot"`
}

type ExtImpData struct {
	PbAdslot string              `json:"pbadslot,omitempty"`
	AdServer *ExtImpDataAdServer `json:"adserver,omitempty"`
}

type ExtImp struct {
	Prebid *openrtb_ext.ExtImpPrebid `json:"prebid,omitempty"`
	Bidder json.RawMessage           `json:"bidder"`
	Data   *ExtImpData               `json:"data,omitempty"`
	Gpid   string                    `json:"gpid,omitempty"`
}

func processImp(imp *openrtb2.Imp) error {
	// get the grid extension
	var ext adapters.ExtImpBidder
	var gridExt openrtb_ext.ExtImpGrid
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal(ext.Bidder, &gridExt); err != nil {
		return err
	}

	if gridExt.Uid == 0 {
		err := &errortypes.BadInput{
			Message: "uid is empty",
		}
		return err
	}
	// no error
	return nil
}

func setImpExtData(imp openrtb2.Imp) openrtb2.Imp {
	var ext ExtImp
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return imp
	}
	if ext.Data != nil && ext.Data.AdServer != nil && ext.Data.AdServer.AdSlot != "" {
		ext.Gpid = ext.Data.AdServer.AdSlot
		extJSON, err := json.Marshal(ext)
		if err == nil {
			imp.Ext = extJSON
		}
	}
	return imp
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *GridAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors = make([]error, 0)

	// copy the request, because we are going to mutate it
	requestCopy := *request
	// this will contain all the valid impressions
	var validImps []openrtb2.Imp
	// pre-process the imps
	for _, imp := range requestCopy.Imp {
		if err := processImp(&imp); err == nil {
			validImps = append(validImps, setImpExtData(imp))
		} else {
			errors = append(errors, err)
		}
	}
	if len(validImps) == 0 {
		err := &errortypes.BadInput{
			Message: "No valid impressions for grid",
		}
		errors = append(errors, err)
		return nil, errors
	}
	requestCopy.Imp = validImps

	reqJSON, err := json.Marshal(requestCopy)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

// MakeBids unpacks the server's response into Bids.
func (a *GridAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				return nil, []error{err}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, nil

}

// Builder builds a new instance of the Grid adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &GridAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}

			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}

			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unknown impression type for ID: \"%s\"", impID),
			}
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to find impression for ID: \"%s\"", impID),
	}
}
