package akcelo

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"net/http"
	"net/url"
)

type adapter struct {
	uri *url.URL
}

type extObj struct {
	Akcelo json.RawMessage `json:"akcelo"`
}

var noValidSiteIdError = &errortypes.BadInput{Message: "Cannot find valid siteId"}
var noValidImpError = &errortypes.BadInput{Message: "No valid Imp"}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	uri, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, err
	}
	bidder := &adapter{uri: uri}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestData, errs := a.prepareBidRequest(request)
	return requestData, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}
	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}
	return extractBids(bidResponse)
}

func (a *adapter) prepareBidRequest(bidRequest *openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	if len(bidRequest.Imp) == 0 {
		return nil, []error{noValidImpError}
	}

	bidRequest = createSitePublisher(bidRequest)
	requestData := make([]*adapters.RequestData, 0, 1)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	var errs []error
	for i := range bidRequest.Imp {
		imp := &bidRequest.Imp[i]
		err := a.prepareImp(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if err := configureParentAccount(bidRequest); err != nil {
		return nil, []error{err}
	}

	var resultBidRequest, err = jsonutil.Marshal(bidRequest)
	if err != nil {
		errs = append(errs, err)
	}

	requestData = append(requestData, &adapters.RequestData{
		Method:  "POST",
		Uri:     a.uri.String(),
		Body:    resultBidRequest,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(bidRequest.Imp),
	})
	return requestData, errs
}

func createSitePublisher(bidRequest *openrtb2.BidRequest) *openrtb2.BidRequest {
	akceloRequest := *bidRequest
	if akceloRequest.Site == nil {
		akceloRequest.Site = &openrtb2.Site{}
	} else {
		akceloRequest.Site = ptrutil.Clone(akceloRequest.Site)
	}
	if akceloRequest.Site.Publisher == nil {
		akceloRequest.Site.Publisher = &openrtb2.Publisher{}
	} else {
		akceloRequest.Site.Publisher = ptrutil.Clone(akceloRequest.Site.Publisher)
	}
	return &akceloRequest
}

func configureParentAccount(bidRequest *openrtb2.BidRequest) error {
	if len(bidRequest.Imp) == 0 {
		return noValidImpError
	}
	parentAccount, _, _, err := jsonparser.Get(bidRequest.Imp[0].Ext, "akcelo", "siteId")
	if err != nil {
		return noValidSiteIdError
	}
	var publisherExt = openrtb_ext.ExtPublisher{}
	publisherExt.Prebid = &openrtb_ext.ExtPublisherPrebid{}
	var parentAccountStr = string(parentAccount)
	publisherExt.Prebid.ParentAccount = &parentAccountStr
	bidRequest.Site.Publisher.Ext, err = jsonutil.Marshal(&publisherExt)
	return err
}

func (a *adapter) prepareImp(imp *openrtb2.Imp) error {
	var extBidder adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &extBidder); err != nil {
		return &errortypes.BadInput{Message: fmt.Sprintf("Unsupported imp : %s", imp.ID)}
	}

	extJson, err := jsonutil.Marshal(extObj{Akcelo: extBidder.Bidder})
	if err != nil {
		return &errortypes.BadInput{Message: fmt.Sprintf("Cannot set akcelo parameters : %s", imp.ID)}
	}
	imp.Ext = extJson
	return nil
}

func extractBids(bidResponse openrtb2.BidResponse) (*adapters.BidderResponse, []error) {
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	var errs []error
	for j := range bidResponse.SeatBid {
		seat := bidResponse.SeatBid[j]
		for i := range seat.Bid {
			bid := seat.Bid[i]
			bidType, err := getBidType(&bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}
	return bidderResponse, errs
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.MType != 0 {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		case openrtb2.MarkupNative:
			return openrtb_ext.BidTypeNative, nil
		default:
			return "", fmt.Errorf("unable to get media type %d", bid.MType)
		}
	}
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
			return "", err
		}
		if bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}
	return "", fmt.Errorf("missing media type for bid: %s", bid.ID)
}
