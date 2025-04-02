package nexx360

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type Nexx360Adapter struct {
	endpoint string
}

type MakeImpOutput struct {
	Imp []openrtb2.Imp
	TagId string
	Placement string
}

type Ext struct {
	Nexx360 ImpNexx360Ext `json:"nexx360"`
}

type ImpNexx360Ext struct {
	TagId string `json:"tagId,omitempty"`
	Placement string `json:"placement,omitempty"`
}

type Nexx360Caller struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type ReqExt struct {
	Nexx360 *ReqNexx360Ext `json:"nexx360,omitempty"`
}

type ReqNexx360Ext struct {
	Caller []Nexx360Caller `json:"caller,omitempty"`
}

type Nexx360ResBidExt struct {
	BidType string `json:"bidType,omitempty"`
}

// CALLER Info used to track Prebid Server
// as one of the hops in the request to exchange
var CALLER = Nexx360Caller{"Prebid-Server", "n/a"}

func makeImps(impList []openrtb2.Imp) (MakeImpOutput, error) {
	var output MakeImpOutput
	var imps []openrtb2.Imp
	var tagId string
	var placement string
	for idx, imp := range impList {
		if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
			return MakeImpOutput{}, &errortypes.BadInput{
				Message: fmt.Sprintf("Imp ID %s must have at least one of [Banner, Video, Audio, Native] defined", imp.ID),
			}
		}
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return MakeImpOutput{}, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var nexx360Ext openrtb_ext.ExtImpNexx360
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &nexx360Ext); err != nil {
			return MakeImpOutput{}, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var impExt Ext
		impExt.Nexx360.TagId = nexx360Ext.TagId
		impExt.Nexx360.Placement = nexx360Ext.Placement


		impExtJSON, err := json.Marshal(impExt)
		if err != nil {
			return MakeImpOutput{}, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		imp.Ext = impExtJSON
		imps = append(imps, imp)
		if idx == 0 {
			tagId = nexx360Ext.TagId 
			placement = nexx360Ext.Placement
		}
		
	}
	output.Imp = imps
	output.TagId = tagId
	output.Placement = placement
	return output, nil
}


func (a *Nexx360Adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var makeImp, err = makeImps(request.Imp);
	if err != nil {
		return nil, []error{err}
	}

	request.Imp = makeImp.Imp
	
	uri := a.endpoint
	if(makeImp.Placement != "") {
		uri = uri + "?placement=" + makeImp.Placement
	}
	if(makeImp.TagId != "") {
		uri = uri + "?tag_id=" + makeImp.TagId
	}


	reqExt, err := makeReqExt(request)
	if err != nil {
		return nil, []error{err}
	}
	request.Ext = reqExt


	// Last Step
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}
	fmt.Printf("Nexx360: Request JSON: %s\n", string(reqJSON))

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	adapter := &adapters.RequestData{
		Method:  "POST",
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	};

	return []*adapters.RequestData{adapter}, nil
}

func makeReqExt(request *openrtb2.BidRequest) ([]byte, error) {
	var reqExt ReqExt

	if len(request.Ext) > 0 {
		if err := jsonutil.Unmarshal(request.Ext, &reqExt); err != nil {
			return nil, err
		}
	}

	if reqExt.Nexx360 == nil {
		reqExt.Nexx360 = &ReqNexx360Ext{}
	}

	if reqExt.Nexx360.Caller == nil {
		reqExt.Nexx360.Caller = make([]Nexx360Caller, 0)
	}

	reqExt.Nexx360.Caller = append(reqExt.Nexx360.Caller, CALLER)

	return json.Marshal(reqExt)
}

// MakeBids make the bids for the bid response.
func (a *Nexx360Adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected http status code: %d", responseData.StatusCode),
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		fmt.Printf("Nexx360: Bad server response: %d\n", responseData.StatusCode)
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected http status code: %d", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse

	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	var Bids []*adapters.TypedBid
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			
			bidType, err := getBidType(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			Bids = append(Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	if len(Bids) == 0 {
		return nil, nil
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(Bids))
	bidResponse.Bids = Bids
	bidResponse.Currency = response.Cur
	
	return bidResponse, errors
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	var bidExt Nexx360ResBidExt
	err := jsonutil.Unmarshal(bid.Ext, &bidExt)
	if(err == nil) {
		if bidExt.BidType == "video" {
			return openrtb_ext.BidTypeVideo, nil
		}
		if bidExt.BidType == "audio" {
			return openrtb_ext.BidTypeAudio, nil
		}
		if (bidExt.BidType == "native") {
			return openrtb_ext.BidTypeNative, nil
		}
		if (bidExt.BidType == "banner") {
			return openrtb_ext.BidTypeBanner, nil
		}
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unable to fetch mediaType in multi-format: %s", bid.ImpID),
	}
}


// Builder builds a new instance of the nexx360 adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &Nexx360Adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
