package stackadapt

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %w", err)
	}
	return &adapter{
		endpoint: endpointTemplate,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	// Imp level
	publisherID, supplyID, err := setImpsAndGetEndpointParams(request)
	if err != nil {
		return nil, []error{err}
	}

	// Request level
	setPublisherID(request, publisherID)

	endpointURL, err := a.buildEndpointURL(publisherID, supplyID)
	if err != nil {
		return nil, []error{err}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, []error{fmt.Errorf("marshal bidRequest: %w", err)}
	}

	return []*adapters.RequestData{{
		Method: http.MethodPost,
		Uri:    endpointURL,
		Body:   body,
		Headers: http.Header{
			"Content-Type": {"application/json;charset=utf-8"},
			"Accept":       {"application/json"},
		},
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

func (a *adapter) buildEndpointURL(publisherID, supplyID string) (string, error) {
	params := macros.EndpointTemplateParams{
		PublisherID: publisherID,
		SupplyId:    supplyID,
	}
	return macros.ResolveMacros(a.endpoint, params)
}

func setImpsAndGetEndpointParams(request *openrtb2.BidRequest) (string, string, error) {
	var publisherID, supplyID string
	for i, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: unable to unmarshal ext: %s", i, err.Error())}
		}

		var saExt openrtb_ext.ExtImpStackAdapt
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &saExt); err != nil {
			return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: unable to unmarshal bidder ext: %s", i, err.Error())}
		}

		if saExt.PublisherId == "" {
			return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: publisherId is required", i)}
		}

		if saExt.SupplyId == "" {
			return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: supplyId is required", i)}
		}

		if publisherID == "" {
			publisherID = saExt.PublisherId
		}
		if supplyID == "" {
			supplyID = saExt.SupplyId
		}

		if saExt.PlacementId != "" {
			request.Imp[i].TagID = saExt.PlacementId
		}

		if saExt.BidFloor > 0 {
			request.Imp[i].BidFloor = saExt.BidFloor
			request.Imp[i].BidFloorCur = "USD"
		}

		if saExt.Banner != nil && len(saExt.Banner.ExpDir) > 0 && request.Imp[i].Banner != nil {
			bannerCopy := *request.Imp[i].Banner
			bannerCopy.ExpDir = make([]adcom1.ExpandableDirection, len(saExt.Banner.ExpDir))
			for j, dir := range saExt.Banner.ExpDir {
				bannerCopy.ExpDir[j] = adcom1.ExpandableDirection(dir)
			}
			request.Imp[i].Banner = &bannerCopy
		}
	}

	return publisherID, supplyID, nil
}

func setPublisherID(request *openrtb2.BidRequest, publisherID string) {
	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = publisherID
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{ID: publisherID}
		}
		request.Site = &siteCopy
	} else if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			publisherCopy.ID = publisherID
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{ID: publisherID}
		}
		request.App = &appCopy
	}
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if bidType == openrtb_ext.BidTypeNative {
				seatBid.Bid[i].AdM, err = getNativeAdm(seatBid.Bid[i].AdM)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
			resolveMacros(&seatBid.Bid[i])

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d", bid.MType),
		}
	}
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid == nil {
		return
	}
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.ReplaceAll(bid.AdM, "${AUCTION_PRICE}", price)
	bid.NURL = strings.ReplaceAll(bid.NURL, "${AUCTION_PRICE}", price)
	bid.BURL = strings.ReplaceAll(bid.BURL, "${AUCTION_PRICE}", price)
}

func getNativeAdm(adm string) (string, error) {
	nativeAdm := make(map[string]interface{})
	if err := jsonutil.Unmarshal([]byte(adm), &nativeAdm); err != nil {
		return adm, errors.New("unable to unmarshal native adm")
	}

	if _, ok := nativeAdm["native"]; ok {
		value, dataType, _, err := jsonparser.Get([]byte(adm), string(openrtb_ext.BidTypeNative))
		if err != nil || dataType != jsonparser.Object {
			return adm, errors.New("unable to get native adm")
		}
		adm = string(value)
	}

	return adm, nil
}
