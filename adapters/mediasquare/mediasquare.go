package mediasquare

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var (
		requestData []*adapters.RequestData
		errs        []error
	)
	if request == nil || request.Imp == nil {
		errs = append(errs, errorWriter("<MakeRequests> request", nil, true))
		return nil, errs
	}

	msqParams := initMsqParams(request)
	msqParams.Test = (request.Test == int8(1))
	for _, imp := range request.Imp {
		var (
			bidderExt   adapters.ExtImpBidder
			msqExt      openrtb_ext.ImpExtMediasquare
			currentCode = msqParametersCodes{
				AdUnit:    imp.TagID,
				AuctionId: request.ID,
				BidId:     imp.ID,
			}
		)

		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, errorWriter("<MakeRequests> imp[ext]", err, len(imp.Ext) == 0))
			continue
		}
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &msqExt); err != nil {
			errs = append(errs, errorWriter("<MakeRequests> imp-bidder[ext]", err, len(bidderExt.Bidder) == 0))
			continue
		}
		currentCode.Owner = msqExt.Owner
		currentCode.Code = msqExt.Code

		if currentCode.setContent(imp) {
			msqParams.Codes = append(msqParams.Codes, currentCode)
		}
	}

	req, err := a.makeRequest(request, &msqParams)
	if err != nil {
		errs = append(errs, err)
	} else if req != nil {
		requestData = append(requestData, req)
	}
	return requestData, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, msqParams *msqParameters) (requestData *adapters.RequestData, err error) {
	var requestJsonBytes []byte
	if msqParams == nil {
		err = errorWriter("<makeRequest> msqParams", nil, true)
		return
	}
	if requestJsonBytes, err = jsonutil.Marshal(msqParams); err == nil {
		var headers http.Header = headerList
		requestData = &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    requestJsonBytes,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		}
	} else {
		err = errorWriter("<makeRequest> jsonutil.Marshal", err, false)
	}

	return
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var (
		bidderResponse *adapters.BidderResponse
		errs           []error
	)
	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case http.StatusBadRequest:
			errs = []error{&errortypes.BadInput{Message: fmt.Sprintf("<MakeBids> Unexpected status code: %d.", response.StatusCode)}}
		default:
			errs = []error{&errortypes.BadServerResponse{
				Message: fmt.Sprintf("<MakeBids> Unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
			}}
		}
		return bidderResponse, errs
	}

	var msqResp msqResponse
	if err := jsonutil.Unmarshal(response.Body, &msqResp); err != nil {
		errs = []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("<MakeBids> Unexpected status code: %d. Bad server response: %s.",
				http.StatusNotAcceptable, err.Error())},
		}
		return bidderResponse, errs
	}
	if len(msqResp.Responses) == 0 {
		errs = []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("<MakeBids> Unexpected status code: %d. No responses found into body content.",
				http.StatusNoContent)},
		}
		return bidderResponse, errs
	}
	bidderResponse = adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	msqResp.getContent(bidderResponse)

	return bidderResponse, errs
}
