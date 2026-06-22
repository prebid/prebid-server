package medianet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpointTemplate *template.Template
	extraInfo        string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	endpoint, err := a.buildEndpointURL(getRegion(request.Imp))
	if err != nil {
		return nil, []error{err}
	}

	reqJson, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     endpoint,
		Body:    reqJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

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

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getBidMediaTypeFromMtype(&sb.Bid[i])
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

// Builder builds a new instance of the Medianet adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{
		endpointTemplate: endpointTemplate,
		extraInfo:        config.ExtraAdapterInfo,
	}, nil
}

func getBidMediaTypeFromMtype(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unable to fetch mediaType for imp: %s", bid.ImpID)
	}
}

func buildEndpoint(mnetUrl, hostUrl string) string {

	if len(hostUrl) == 0 {
		return mnetUrl
	}
	urlObject, err := url.Parse(mnetUrl)
	if err != nil {
		return mnetUrl
	}
	values := urlObject.Query()
	values.Add("src", hostUrl)
	urlObject.RawQuery = values.Encode()
	return urlObject.String()
}

// buildEndpointURL resolves the Host macro in the endpoint template using the
// host that corresponds to the provided region and appends the configured src param.
func (a *adapter) buildEndpointURL(region string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: getRegionHost(region)}
	endpoint, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)
	if err != nil {
		return "", err
	}
	return buildEndpoint(endpoint, a.extraInfo), nil
}

// getRegion returns the Media.net region from the first impression's bidder
// params. An empty string is returned when no region is provided.
func getRegion(imps []openrtb2.Imp) string {
	if len(imps) == 0 {
		return ""
	}
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imps[0].Ext, &bidderExt); err != nil {
		return ""
	}
	var medianetExt openrtb_ext.ExtImpMedianet
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &medianetExt); err != nil {
		return ""
	}
	return medianetExt.Region
}

// getRegionHost maps a Media.net region to its regional endpoint host,
// falling back to the default host when the region is empty or unknown.
func getRegionHost(region string) string {
	switch strings.ToUpper(region) {
	case "USE":
		return "prebid-adapter-useast.media.net"
	case "USW":
		return "prebid-adapter-uswest.media.net"
	case "APAC":
		return "prebid-adapter-asia.media.net"
	case "EUC":
		return "prebid-adapter-eu.media.net"
	default:
		return "prebid-adapter.media.net"
	}
}
