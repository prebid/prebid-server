package smartrtb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// Base adapter structure.
type SmartRTBAdapter struct {
	EndpointTemplate *template.Template
}

// Bid request extension appended to downstream request.
// PubID are non-empty iff request.{App,Site} or
// request.{App,Site}.Publisher are nil, respectively.
type bidRequestExt struct {
	PubID    string `json:"pub_id,omitempty"`
	ZoneID   string `json:"zone_id,omitempty"`
	ForceBid bool   `json:"force_bid,omitempty"`
}

// bidExt.CreativeType values.
// nolint: staticcheck // staticcheck SA9004: only the first constant in this group has an explicit type
const (
	creativeTypeBanner string = "BANNER"
	creativeTypeVideo         = "VIDEO"
	creativeTypeNative        = "NATIVE"
	creativeTypeAudio         = "AUDIO"
)

// Bid response extension from downstream.
type bidExt struct {
	CreativeType string `json:"format"`
}

// Builder builds a new instance of the SmartRTB adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &SmartRTBAdapter{
		EndpointTemplate: template,
	}
	return bidder, nil
}

func (adapter *SmartRTBAdapter) buildEndpointURL(pubID string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: pubID}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)
}

func parseExtImp(dst *bidRequestExt, imp *openrtb2.Imp) error {
	var ext adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var src openrtb_ext.ExtImpSmartRTB
	if err := jsonutil.Unmarshal(ext.Bidder, &src); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	if dst.PubID == "" {
		dst.PubID = src.PubID
	}

	if src.ZoneID != "" {
		imp.TagID = src.ZoneID
	}
	return nil
}

func (s *SmartRTBAdapter) MakeRequests(brq *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var imps []openrtb2.Imp
	var err error
	ext := bidRequestExt{}
	nrImps := len(brq.Imp)
	errs := make([]error, 0, nrImps)

	for i := 0; i < nrImps; i++ {
		imp := brq.Imp[i]
		if imp.Banner == nil && imp.Video == nil {
			continue
		}

		err = parseExtImp(&ext, &imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		imps = append(imps, imp)
	}

	if len(imps) == 0 {
		return nil, errs
	}

	if ext.PubID == "" {
		return nil, append(errs, &errortypes.BadInput{Message: "Cannot infer publisher ID from bid ext"})
	}

	brq.Ext, err = json.Marshal(ext)
	if err != nil {
		return nil, append(errs, err)
	}

	brq.Imp = imps

	rq, err := json.Marshal(brq)
	if err != nil {
		return nil, append(errs, err)
	}

	url, err := s.buildEndpointURL(ext.PubID)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     url,
		Body:    rq,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(brq.Imp),
	}}, errs
}

func (s *SmartRTBAdapter) MakeBids(
	brq *openrtb2.BidRequest, drq *adapters.RequestData,
	rs *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if rs.StatusCode == http.StatusNoContent {
		return nil, nil
	} else if rs.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{Message: "Invalid request."}}
	} else if rs.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected HTTP status %d.", rs.StatusCode),
		}}
	}

	var brs openrtb2.BidResponse
	if err := jsonutil.Unmarshal(rs.Body, &brs); err != nil {
		return nil, []error{err}
	}

	rv := adapters.NewBidderResponseWithBidsCapacity(5)
	for _, seat := range brs.SeatBid {
		for i := range seat.Bid {
			var ext bidExt
			if err := jsonutil.Unmarshal(seat.Bid[i].Ext, &ext); err != nil {
				return nil, []error{&errortypes.BadServerResponse{
					Message: "Invalid bid extension from endpoint.",
				}}
			}

			var btype openrtb_ext.BidType
			switch ext.CreativeType {
			case creativeTypeBanner:
				btype = openrtb_ext.BidTypeBanner
			case creativeTypeVideo:
				btype = openrtb_ext.BidTypeVideo
			default:
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unsupported creative type %s.",
						ext.CreativeType),
				}}
			}

			seat.Bid[i].Ext = nil

			rv.Bids = append(rv.Bids, &adapters.TypedBid{
				Bid:     &seat.Bid[i],
				BidType: btype,
			})
		}
	}
	return rv, nil
}
