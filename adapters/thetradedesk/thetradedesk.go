package thetradedesk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"text/template"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type adapter struct {
	bidderEndpointTemplate string
	defaultEndpoint        string
	templateEndpoint       *template.Template
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	pubID, supplySourceId, err := getExtensionInfo(request.Imp)

	if err != nil {
		return nil, []error{err}
	}

	modifiedImps := make([]openrtb2.Imp, 0, len(request.Imp))

	for _, imp := range request.Imp {

		if imp.Banner != nil {
			if len(imp.Banner.Format) > 0 {
				firstFormat := imp.Banner.Format[0]
				bannerCopy := *imp.Banner
				bannerCopy.H = &firstFormat.H
				bannerCopy.W = &firstFormat.W
				imp.Banner = &bannerCopy

			}
		}

		modifiedImps = append(modifiedImps, imp)
	}

	request.Imp = modifiedImps

	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			if pubID != "" {
				publisherCopy.ID = pubID
			}
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.Site = &siteCopy
	} else if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			if pubID != "" {
				publisherCopy.ID = pubID
			}
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.App = &appCopy
	}

	errs := make([]error, 0, len(request.Imp))
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	bidderEndpoint, err := a.buildEndpointURL(supplySourceId)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     bidderEndpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

func (a *adapter) buildEndpointURL(supplySourceId string) (string, error) {
	if supplySourceId == "" {
		if a.defaultEndpoint == "" {
			return "", errors.New("Either supplySourceId or a default endpoint must be provided")
		}
		return a.defaultEndpoint, nil
	}

	urlParams := macros.EndpointTemplateParams{SupplyId: supplySourceId}
	bidderEndpoint, err := macros.ResolveMacros(a.templateEndpoint, urlParams)

	if err != nil {
		return "", fmt.Errorf("unable to resolve endpoint macros: %v", err)
	}

	return bidderEndpoint, nil
}

func getExtensionInfo(impressions []openrtb2.Imp) (string, string, error) {
	publisherId := ""
	supplySourceId := ""
	for _, imp := range impressions {
		var ttdExt, err = getImpressionExt(&imp)
		if err != nil {
			return "", "", err
		}

		if ttdExt.PublisherId != "" && publisherId == "" {
			publisherId = ttdExt.PublisherId
		}

		if ttdExt.SupplySourceId != "" && supplySourceId == "" {
			supplySourceId = ttdExt.SupplySourceId
		}

		if publisherId != "" && supplySourceId != "" {
			break
		}
	}
	return publisherId, supplySourceId, nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpTheTradeDesk, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}

	var ttdExt openrtb_ext.ExtImpTheTradeDesk
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &ttdExt); err != nil {
		return nil, err
	}
	return &ttdExt, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return adapters.NewBidderResponse(), nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = bidResponse.Cur

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid

			bidType, err := getBidType(bid.MType)

			if err != nil {
				return nil, []error{err}
			}

			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			bidderResponse.Bids = append(bidderResponse.Bids, b)
		}
	}

	return bidderResponse, nil
}

func getBidType(markupType openrtb2.MarkupType) (openrtb_ext.BidType, error) {
	switch markupType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unsupported mtype: %d", markupType)
	}
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	if len(config.ExtraAdapterInfo) > 0 {
		isValidEndpoint, err := regexp.Match("([a-z]+)$", []byte(config.ExtraAdapterInfo))
		if !isValidEndpoint || err != nil {
			return nil, errors.New("ExtraAdapterInfo must be a simple string provided by TheTradeDesk")
		}
	}

	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	urlParams := macros.EndpointTemplateParams{SupplyId: config.ExtraAdapterInfo}
	defaultEndpoint, err := macros.ResolveMacros(template, urlParams)

	if err != nil {
		return nil, fmt.Errorf("unable to resolve endpoint macros: %v", err)
	}

	return &adapter{
		bidderEndpointTemplate: config.Endpoint,
		defaultEndpoint:        defaultEndpoint,
		templateEndpoint:       template,
	}, nil
}
