package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"

	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

const defaultNetwork = "default"
const defaultSite = "unknown"
const minutesInHour = 60

// Builder builds a new instance of the Adnuntius adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		time:      &timeutil.RealTime{},
		endpoint:  config.Endpoint,
		extraInfo: config.ExtraAdapterInfo,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return a.generateRequests(*request)
}

func (a *adapter) generateRequests(ortbRequest openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	var requestData []*adapters.RequestData
	networkAdunitMap := make(map[string][]adnRequestAdunit)
	headers := setHeaders(ortbRequest)
	var noCookies bool = false

	for _, imp := range ortbRequest.Imp {

		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
			}}
		}
		var adnuntiusExt openrtb_ext.ImpExtAdnunitus
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpValues: %s", err.Error()),
			}}
		}

		if adnuntiusExt.NoCookies {
			noCookies = true
		}

		network := defaultNetwork
		if adnuntiusExt.Network != "" {
			network = adnuntiusExt.Network
		}

		// Remove when we support video.
		if imp.Video != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("ignoring imp id=%s, Adnuntius supports only native and banner", imp.ID),
			}}
		}

		if imp.Banner != nil {
			adUnit := generateAdUnit(imp, adnuntiusExt, "banner")
			adUnit.AdType = ""

			networkAdunitMap[network] = append(
				networkAdunitMap[network],
				adUnit)
		}

		if imp.Native != nil {
			adUnit := generateAdUnit(imp, adnuntiusExt, "native")
			adUnit.AdType = "NATIVE"
			nativeRequest := json.RawMessage{}

			if err := jsonutil.Unmarshal([]byte(imp.Native.Request), &nativeRequest); err != nil {
				return nil, []error{&errortypes.BadInput{
					Message: fmt.Sprintf("Error unmarshalling Native: %s", err.Error()),
				}}
			}

			adUnit.NativeRequest.Ortb = nativeRequest
			networkAdunitMap[network] = append(
				networkAdunitMap[network],
				adUnit)
		}

	}

	endpoint, err := makeEndpointUrl(ortbRequest, a, noCookies)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("failed to parse URL: %s", err),
		}}
	}

	site := defaultSite
	if ortbRequest.Site != nil && ortbRequest.Site.Page != "" {
		site = ortbRequest.Site.Page
	}

	extSite, err := getSiteExtAsKv(&ortbRequest)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to parse site Ext: %v", err)}
	}

	var extUser openrtb_ext.ExtUser
	if ortbRequest.User != nil && ortbRequest.User.Ext != nil {
		if err := jsonutil.Unmarshal(ortbRequest.User.Ext, &extUser); err != nil {
			return nil, []error{fmt.Errorf("failed to parse Ext User: %v", err)}
		}
	}

	for _, networkAdunits := range networkAdunitMap {

		adnuntiusRequest := adnRequest{
			AdUnits:   networkAdunits,
			Context:   site,
			KeyValues: extSite.Data,
		}

		// Will change when our adserver can accept multiple user IDS
		if extUser.Eids != nil && len(extUser.Eids) > 0 {
			if len(extUser.Eids[0].UIDs) > 0 {
				adnuntiusRequest.MetaData.Usi = extUser.Eids[0].UIDs[0].ID
			}
		}

		ortbUser := ortbRequest.User
		if ortbUser != nil {
			ortbUserId := ortbRequest.User.ID
			if ortbUserId != "" {
				adnuntiusRequest.MetaData.Usi = ortbRequest.User.ID
			}
		}

		adnJson, err := json.Marshal(adnuntiusRequest)
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling adnuntius request: %s", err.Error()),
			}}
		}

		requestData = append(requestData, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    adnJson,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(ortbRequest.Imp),
		})

	}

	return requestData, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Status code: %d, Request malformed", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Status code: %d, Something went wrong with your request", response.StatusCode),
		}}
	}

	var adnResponse AdnResponse
	if err := jsonutil.Unmarshal(response.Body, &adnResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse, bidErr := generateBidResponse(&adnResponse, request)
	if bidErr != nil {
		return nil, bidErr
	}

	return bidResponse, nil
}

func generateBidResponse(adnResponse *AdnResponse, request *openrtb2.BidRequest) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adnResponse.AdUnits))
	var currency string
	adunitMap := map[string]AdUnit{}
	adunitMediaTypeMap := map[string][]AdUnit{}

	/* Check the ad unit response to see if there are any multi ad  */
	for _, adnRespAdunit := range adnResponse.AdUnits {
		result := strings.Split(adnRespAdunit.TargetId, ":")
		if adnRespAdunit.MatchedAdCount > 0 {
			adunitMediaTypeMap[result[0]] = append(adunitMediaTypeMap[result[0]], adnRespAdunit)
		}
	}

	/* Compare price if there are multiple media types */
	for targetId, mappedAdunit := range adunitMediaTypeMap {
		highestBidAtIndex := 0
		if len(mappedAdunit) > 1 {
			for index := range mappedAdunit {
				if mappedAdunit[index].Ads[0].Bid.Amount > mappedAdunit[highestBidAtIndex].Ads[0].Bid.Amount {
					highestBidAtIndex = index
				}
			}
		}
		adunitMap[targetId] = mappedAdunit[highestBidAtIndex]
	}

	for _, imp := range request.Imp {

		auId, _, _, err := jsonparser.Get(imp.Ext, "bidder", "auId")
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error at Bidder auId: %s", err.Error()),
			}}
		}

		targetID := fmt.Sprintf("%s-%s", string(auId), imp.ID)

		adunit := adunitMap[targetID]

		if len(adunit.Ads) > 0 {

			ad := adunit.Ads[0]
			html := adunit.Html
			var mType openrtb2.MarkupType = openrtb2.MarkupBanner
			var native []byte

			currency = ad.Bid.Currency
			if adunit.NativeJson != nil {
				nativeJson, _, _, nativeErr := jsonparser.Get(adunit.NativeJson, "ortb")
				if nativeErr != nil {
					return nil, []error{&errortypes.BadServerResponse{
						Message: fmt.Sprintf("Failed to parse native json where imp id=%s", imp.ID),
					}}
				}
				native = nativeJson
			}

			if native != nil {
				html = string(native)
				mType = openrtb2.MarkupNative
			}

			adBid, err := generateAdResponse(ad, imp, html, mType, request)
			if err != nil {
				return nil, err
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     adBid,
				BidType: convertMarkupTypeToBidType(mType),
			})

			for _, deal := range adunit.Deals {
				mType = 1
				dealBid, err := generateAdResponse(deal, imp, deal.Html, mType, request)
				if err != nil {
					return nil, err
				}

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     dealBid,
					BidType: convertMarkupTypeToBidType(mType),
				})
			}
		}
	}
	bidResponse.Currency = currency
	return bidResponse, nil
}

func generateAdResponse(ad Ad, imp openrtb2.Imp, html string, mType openrtb2.MarkupType, request *openrtb2.BidRequest) (*openrtb2.Bid, []error) {
	creativeWidth, widthErr := strconv.ParseInt(ad.CreativeWidth, 10, 64)
	if widthErr != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Value of width: %s is not a string", ad.CreativeWidth),
		}}
	}

	creativeHeight, heightErr := strconv.ParseInt(ad.CreativeHeight, 10, 64)
	if heightErr != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Value of height: %s is not a string", ad.CreativeHeight),
		}}
	}

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
		}}
	}

	var adnuntiusExt openrtb_ext.ImpExtAdnunitus
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error unmarshalling ExtImpValues: %s", err.Error()),
		}}
	}

	price := ad.Bid.Amount
	if adnuntiusExt.BidType != "" {
		if strings.EqualFold(string(adnuntiusExt.BidType), "net") {
			price = ad.NetBid.Amount
		}
		if strings.EqualFold(string(adnuntiusExt.BidType), "gross") {
			price = ad.GrossBid.Amount
		}
	}

	extJson, err := generateReturnExt(ad, request)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error extracting Ext: %s", err.Error()),
		}}
	}

	bid := openrtb2.Bid{
		ID:      ad.AdId,
		ImpID:   imp.ID,
		W:       creativeWidth,
		H:       creativeHeight,
		AdID:    ad.AdId,
		DealID:  ad.DealID,
		CID:     ad.LineItemId,
		CrID:    ad.CreativeId,
		Price:   price * 1000,
		AdM:     html,
		MType:   mType,
		ADomain: ad.AdvertiserDomains,
		Ext:     extJson,
	}

	return &bid, nil
}
