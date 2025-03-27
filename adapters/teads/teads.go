package teads

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// Builder builds a new instance of the Teads adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpointTemplate: template,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	endpointURL, err := a.buildEndpointURL()
	if endpointURL == "" {
		return nil, []error{err}
	}

	if err := updateImpObject(request.Imp); err != nil {
		return nil, []error{err}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "Error parsing BidRequest object",
		}}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     endpointURL,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, []error{}
}

func updateImpObject(imps []openrtb2.Imp) error {
	for i := range imps {
		imp := &imps[i]

		if imp.Banner != nil {
			if len(imp.Banner.Format) != 0 {
				bannerCopy := *imp.Banner
				bannerCopy.H = &imp.Banner.Format[0].H
				bannerCopy.W = &imp.Banner.Format[0].W
				imp.Banner = &bannerCopy
			}
		}

		var defaultImpExt defaultBidderImpExtension
		if err := jsonutil.Unmarshal(imp.Ext, &defaultImpExt); err != nil {
			return &errortypes.BadInput{
				Message: "Error parsing Imp.Ext object",
			}
		}
		if defaultImpExt.Bidder.PlacementId == 0 {
			return &errortypes.BadInput{
				Message: "placementId should not be 0.",
			}
		}
		imp.TagID = strconv.Itoa(defaultImpExt.Bidder.PlacementId)
		teadsImpExt := &teadsImpExtension{
			KV: teadsKV{
				PlacementId: defaultImpExt.Bidder.PlacementId,
			},
		}
		if extJson, err := json.Marshal(teadsImpExt); err != nil {
			return &errortypes.BadInput{
				Message: "Error stringify Imp.Ext object",
			}
		} else {
			imp.Ext = extJson
		}
	}
	return nil
}

// Builds enpoint url based on adapter-specific pub settings from imp.ext
func (a *adapter) buildEndpointURL() (string, error) {
	endpointParams := macros.EndpointTemplateParams{}
	host, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)

	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Unable to parse endpoint url template: " + err.Error(),
		}
	}

	thisURI, err := url.Parse(host)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Malformed URL: " + err.Error(),
		}
	}

	return thisURI.String(), nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	err := adapters.CheckResponseStatusCodeForErrors(response)
	if err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid))

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]

			bidExtTeads, err := getTeadsRendererFromBidExt(bid.Ext)
			if err != nil {
				return nil, err
			}
			bidType, err := getMediaTypeForImp(bid.ImpID, internalRequest.Imp)
			if err != nil {
				return nil, err
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid: &bid,
				BidMeta: &openrtb_ext.ExtBidPrebidMeta{
					RendererName:    bidExtTeads.Prebid.Meta.RendererName,
					RendererVersion: bidExtTeads.Prebid.Meta.RendererVersion,
				},
				BidType: bidType,
			})
		}
	}
	if bidResp.Cur != "" {
		bidderResponse.Currency = bidResp.Cur
	}
	return bidderResponse, nil
}

func getTeadsRendererFromBidExt(ext json.RawMessage) (*teadsBidExt, []error) {
	var bidExtTeads teadsBidExt
	if err := jsonutil.Unmarshal(ext, &bidExtTeads); err != nil {
		return nil, []error{err}
	}
	if bidExtTeads.Prebid.Meta.RendererName == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "RendererName should not be empty if present",
		}}
	}
	if bidExtTeads.Prebid.Meta.RendererVersion == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "RendererVersion should not be empty if present",
		}}
	}
	return &bidExtTeads, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, []error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
			return openrtb_ext.BidTypeBanner, nil
		}
	}
	return openrtb_ext.BidType(""), []error{&errortypes.BadInput{
		Message: "Imp ids were not equals",
	}}
}
