package adquery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	defaultCurrency string = "PLN"
	bidderName      string = "adquery"
	prebidVersion   string = "server"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Adquery adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) buildRequest(bidReq *openrtb2.BidRequest, imp *openrtb2.Imp, ext *openrtb_ext.ImpExtAdQuery) *BidderRequest {
	userId := ""
	if bidReq.User != nil {
		userId = bidReq.User.ID
	}

	return &BidderRequest{
		V:                   prebidVersion,
		PlacementCode:       ext.PlacementID,
		AuctionId:           "",
		BidType:             ext.Type,
		AdUnitCode:          imp.TagID,
		BidQid:              userId,
		BidId:               fmt.Sprintf("%s%s", bidReq.ID, imp.ID),
		Bidder:              bidderName,
		BidderRequestId:     bidReq.ID,
		BidRequestsCount:    1,
		BidderRequestsCount: 1,
		Sizes:               getImpSizes(imp),
	}
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, nil
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	var result []*adapters.RequestData
	var errs []error
	for _, imp := range request.Imp {
		ext, err := parseExt(imp.Ext)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{err.Error()})
			continue
		}

		requestJSON, err := json.Marshal(a.buildRequest(request, &imp, ext))
		if err != nil {
			return nil, []error{err}
		}

		data := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    requestJSON,
			Headers: headers,
		}
		result = append(result, data)
	}

	return result, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	respData, price, width, height, errs := parseResponseJson(responseData.Body)
	if errs != nil && len(errs) > 0 {
		return nil, errs
	}

	if respData == nil {
		return adapters.NewBidderResponse(), nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	if respData.Currency != "" {
		bidResponse.Currency = respData.Currency
	} else {
		bidResponse.Currency = defaultCurrency
	}
	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid: &openrtb2.Bid{
			// There's much more possible fields to be added here, see OpenRTB docs for reference (type: Bid)
			ID:      respData.ReqID,
			ImpID:   request.Imp[0].ID,
			Price:   price,
			AdM:     fmt.Sprintf("<script src=\"%s\"></script>%s", respData.AdQLib, respData.Tag),
			ADomain: respData.ADomains,
			CrID:    fmt.Sprintf("%d", respData.CrID),
			W:       width,
			H:       height,
		},
		BidType: respData.MediaType.Name,
	})

	return bidResponse, nil
}

func parseExt(ext json.RawMessage) (*openrtb_ext.ImpExtAdQuery, error) {
	var bext adapters.ExtImpBidder
	err := json.Unmarshal(ext, &bext)
	if err != nil {
		return nil, err
	}

	var adsExt openrtb_ext.ImpExtAdQuery
	err = json.Unmarshal(bext.Bidder, &adsExt)
	if err != nil {
		return nil, err
	}

	// not validating, because it should have been done earlier by the server
	return &adsExt, nil
}

func parseResponseJson(respBody []byte) (*ResponseData, float64, int64, int64, []error) {
	var response ResponseAdQuery
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, 0, 0, 0, []error{err}
	}

	if response.Data == nil {
		return nil, 0, 0, 0, nil
	}

	var errs []error
	price, err := strconv.ParseFloat(response.Data.CPM, 64)
	if err != nil {
		errs = append(errs, err)
	}
	width, err := strconv.ParseInt(response.Data.MediaType.Width, 10, 64)
	if err != nil {
		errs = append(errs, err)
	}
	height, err := strconv.ParseInt(response.Data.MediaType.Height, 10, 64)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, 0, 0, 0, errs
	}
	return response.Data, price, width, height, nil
}

func getImpSizes(imp *openrtb2.Imp) string {
	if imp.Banner == nil {
		return ""
	}

	if len(imp.Banner.Format) > 0 {
		sizes := make([]string, len(imp.Banner.Format))
		for i, format := range imp.Banner.Format {
			sizes[i] = strconv.FormatInt(format.W, 10) + "x" + strconv.FormatInt(format.H, 10)
		}

		return strings.Join(sizes, ",")
	}

	if imp.Banner.W != nil && imp.Banner.H != nil {
		return strconv.FormatInt(*imp.Banner.W, 10) + "x" + strconv.FormatInt(*imp.Banner.H, 10)
	}

	return ""
}
