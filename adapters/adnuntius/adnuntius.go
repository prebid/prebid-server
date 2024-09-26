package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"

	// nativeRequests "github.com/prebid/openrtb/v20/native1/request"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/timeutil"
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
	if err := json.Unmarshal(response.Body, &adnResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse, bidErr := generateBidResponse(&adnResponse, request)
	if bidErr != nil {
		return nil, bidErr
	}

	return bidResponse, nil
}
