package nextmillennium

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
	"github.com/prebid/prebid-server/v3/version"
)

const NM_ADAPTER_VERSION = "v1.0.0"

type adapter struct {
	endpoint string
	nmmFlags []string
	server   config.Server
}

type nmExtPrebidStoredRequest struct {
	ID string `json:"id"`
}

type server struct {
	ExternalUrl string `json:"externalurl"`
	GvlID       int    `json:"gvlid"`
	DataCenter  string `json:"datacenter"`
}
type nmExtPrebid struct {
	StoredRequest nmExtPrebidStoredRequest `json:"storedrequest"`
	Server        *server                  `json:"server,omitempty"`
}
type nmExtNMM struct {
	NmmFlags       []string `json:"nmmFlags,omitempty"`
	ServerVersion  string   `json:"server_version,omitempty"`
	AdapterVersion string   `json:"nm_version,omitempty"`
}
type nextMillJsonExt struct {
	Prebid         nmExtPrebid `json:"prebid"`
	NextMillennium nmExtNMM    `json:"nextMillennium,omitempty"`
}

// MakeRequests prepares request information for prebid-server core
func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	resImps, err := getImpressionsInfo(request.Imp)
	if len(err) > 0 {
		return nil, err
	}

	result := make([]*adapters.RequestData, 0, len(resImps))
	for _, imp := range resImps {
		bidRequest, err := adapter.buildAdapterRequest(request, imp)
		if err != nil {
			return nil, []error{err}
		}
		result = append(result, bidRequest)
	}

	return result, nil
}

func getImpressionsInfo(imps []openrtb2.Imp) (resImps []*openrtb_ext.ImpExtNextMillennium, errors []error) {
	for _, imp := range imps {
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		resImps = append(resImps, impExt)
	}

	return
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtNextMillennium, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var nextMillenniumExt openrtb_ext.ImpExtNextMillennium
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &nextMillenniumExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	return &nextMillenniumExt, nil
}

func (adapter *adapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ImpExtNextMillennium) (*adapters.RequestData, error) {
	newBidRequest := createBidRequest(prebidBidRequest, params, adapter.nmmFlags, adapter.server)

	reqJSON, err := json.Marshal(newBidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(newBidRequest.Imp)}, nil
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ImpExtNextMillennium, flags []string, serverParams config.Server) *openrtb2.BidRequest {
	placementID := params.PlacementID

	if params.GroupID != "" {
		domain := ""
		size := ""

		if prebidBidRequest.Site != nil {
			domain = prebidBidRequest.Site.Domain
		}
		if prebidBidRequest.App != nil {
			domain = prebidBidRequest.App.Domain
		}

		if banner := prebidBidRequest.Imp[0].Banner; banner != nil {
			if len(banner.Format) > 0 {
				size = fmt.Sprintf("%dx%d", banner.Format[0].W, banner.Format[0].H)
			} else if banner.W != nil && banner.H != nil {
				size = fmt.Sprintf("%dx%d", *banner.W, *banner.H)
			}
		}

		placementID = fmt.Sprintf("g%s;%s;%s", params.GroupID, size, domain)
	}
	ext := nextMillJsonExt{}
	ext.Prebid.StoredRequest.ID = placementID
	ext.NextMillennium.NmmFlags = flags
	bidRequest := *prebidBidRequest
	jsonExtCommon, err := json.Marshal(ext)
	if err != nil {
		return prebidBidRequest
	}
	bidRequest.Imp[0].Ext = jsonExtCommon
	ext.Prebid.Server = &server{
		GvlID:       serverParams.GvlID,
		DataCenter:  serverParams.DataCenter,
		ExternalUrl: serverParams.ExternalUrl,
	}
	ext.NextMillennium.AdapterVersion = NM_ADAPTER_VERSION
	ext.NextMillennium.ServerVersion = version.Ver
	jsonExt, err := json.Marshal(ext)
	if err != nil {
		return &bidRequest
	}
	bidRequest.Ext = jsonExt
	return &bidRequest
}

// MakeBids translates NextMillennium bid response to prebid-server specific format
func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var msg = ""
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		msg = fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}

	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		msg = fmt.Sprintf("Bad server response: %d", err)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	var errors []error
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getBidType(sb.Bid[i].MType)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, errors
}

// Builder builds a new instance of the NextMillennium adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	var info nmExtNMM
	if config.ExtraAdapterInfo != "" {
		if err := jsonutil.Unmarshal([]byte(config.ExtraAdapterInfo), &info); err != nil {
			return nil, fmt.Errorf("invalid extra info: %v", err)
		}
	}

	return &adapter{
		endpoint: config.Endpoint,
		nmmFlags: info.NmmFlags,
		server:   server,
	}, nil
}

func getBidType(mType openrtb2.MarkupType) (openrtb_ext.BidType, error) {
	switch mType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{Message: fmt.Sprintf("Unsupported return mType: %v", mType)}
	}
}
