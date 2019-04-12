package sharethrough

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

const hbSource = "prebid-server"
const strVersion = "1.0.0"

func NewSharethroughBidder(endpoint string) *SharethroughAdapter {
	return &SharethroughAdapter{URI: endpoint}
}

type SharethroughAdapter struct {
	URI string
}

// Name returns the adapter name as a string
func (s SharethroughAdapter) Name() string {
	return "sharethrough"
}

func (s SharethroughAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	//fmt.Printf("in sharethrough adapter\nrequest: %+v\n", request)
	errs := make([]error, 0, len(request.Imp))
	headers := http.Header{}
	var potentialRequests []*adapters.RequestData

	headers.Add("Content-Type", "text/plain;charset=utf-8")
	headers.Add("Accept", "application/json")

	for i := 0; i < len(request.Imp); i++ {
		imp := request.Imp[i]

		fmt.Printf("processing imp")

		var extBtlrParams openrtb_ext.ExtImpSharethroughExt
		if err := json.Unmarshal(imp.Ext, &extBtlrParams); err != nil {
			return nil, []error{err}
		}

		var gdprApplies int64 = 0
		if request.Regs != nil {
			if jsonExtRegs, err := request.Regs.Ext.MarshalJSON(); err == nil {
				gdprApplies, _ = jsonparser.GetInt(jsonExtRegs, "gdpr")
			}
		}

		consentString := ""
		if request.User != nil {
			if jsonExtUser, err := request.User.Ext.MarshalJSON(); err == nil {
				consentString, _ = jsonparser.GetString(jsonExtUser, "consent")
			}
		}

		pKey := extBtlrParams.Bidder.Pkey

		var height, width uint64
		if len(extBtlrParams.Bidder.IframeSize) >= 2 {
			height, width = uint64(extBtlrParams.Bidder.IframeSize[0]), uint64(extBtlrParams.Bidder.IframeSize[1])
		} else {
			height, width = getPlacementSize(imp.Banner.Format)
		}

		potentialRequests = append(potentialRequests, &adapters.RequestData{
			Method: "POST",
			Uri: generateHBUri(s.URI, hbUriParams{
				Pkey:               pKey,
				BidID:              imp.ID,
				ConsentRequired:    !(gdprApplies == 0),
				ConsentString:      consentString,
				Iframe:             extBtlrParams.Bidder.Iframe,
				Height:             height,
				Width:              width,
				InstantPlayCapable: canAutoPlayVideo(request.Device.UA),
			}, request.App),
			Body:    nil,
			Headers: headers,
		})
	}

	return potentialRequests, errs
}

func (s SharethroughAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var strBidResp openrtb_ext.ExtImpSharethroughResponse
	if err := json.Unmarshal(response.Body, &strBidResp); err != nil {
		return nil, []error{err}
	}

	br, bidderResponseErr := butlerToOpenRTBResponse(externalRequest, strBidResp)

	return br, bidderResponseErr
}
