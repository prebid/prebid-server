package akcelo

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"net/url"
)

type adapter struct {
	uri url.URL
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	uri, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, err
	}
	bidder := &adapter{uri: *uri}
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

func createSitePublisher(bidRequest *openrtb2.BidRequest) *openrtb2.BidRequest {
	akceloRequest := *bidRequest
	if akceloRequest.Site == nil {
		akceloRequest.Site = &openrtb2.Site{}
	} else {
		siteCopy := *akceloRequest.Site
		akceloRequest.Site = &siteCopy
	}
	if akceloRequest.Site.Publisher == nil {
		akceloRequest.Site.Publisher = &openrtb2.Publisher{}
	} else {
		publisherCopy := *akceloRequest.Site.Publisher
		akceloRequest.Site.Publisher = &publisherCopy
	}
	return &akceloRequest
}

func (a *adapter) prepareBidRequest(bidRequest *openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	bidRequest = createSitePublisher(bidRequest)
	requestData := make([]*adapters.RequestData, 0, 1)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	var errs []error
	for i, imp := range bidRequest.Imp {
		newImp, err := a.prepareImp(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if newImp != nil {
			bidRequest.Imp[i] = *newImp
		}
	}

	if len(bidRequest.Imp) > 0 {
		parentAccount := gjson.Get(string(bidRequest.Imp[0].Ext), "akcelo.siteId").String()
		var publisherExt = openrtb_ext.ExtPublisher{}
		publisherExt.Prebid = &openrtb_ext.ExtPublisherPrebid{}
		publisherExt.Prebid.ParentAccount = &parentAccount
		bidRequest.Site.Publisher.Ext, _ = json.Marshal(&publisherExt)
	} else {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("No valid Imp")}}
	}

	var resultBidRequest, err = json.Marshal(bidRequest)
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

func (a *adapter) prepareImp(imp openrtb2.Imp) (*openrtb2.Imp, error) {
	var extBidder adapters.ExtImpBidder
	err := jsonutil.Unmarshal(imp.Ext, &extBidder)
	if err == nil {
		var out, err = sjson.Set(string(imp.Ext), "akcelo", extBidder.Bidder)
		if err != nil {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("Cannot set akcelo parameters : %s", imp.ID)}
		}
		out, _ = sjson.Delete(out, "bidder")
		imp.Ext = json.RawMessage(out)
		return &imp, nil
	} else {
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("Unsupported imp : %s", imp.ID)}
	}
}

func extractBids(bidResponse openrtb2.BidResponse) (*adapters.BidderResponse, []error) {
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	var errs []error
	for _, seat := range bidResponse.SeatBid {
		for _, bid := range seat.Bid {
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
		case 1:
			return openrtb_ext.BidTypeBanner, nil
		case 2:
			return openrtb_ext.BidTypeVideo, nil
		case 4:
			return openrtb_ext.BidTypeNative, nil
		default:
			return "", fmt.Errorf("unable to fetch media type %d", bid.MType)
		}
	}
	var bidExt openrtb_ext.ExtBid
	err := json.Unmarshal(bid.Ext, &bidExt)
	if err != nil {
		return "", err
	}
	if bidExt.Prebid != nil {
		return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
	}
	return "", fmt.Errorf("missing media type for bid: %s", bid.ID)
}
