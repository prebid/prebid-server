package adquery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
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
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := buildHeaders(request)

	var result []*adapters.RequestData
	var errs []error
	for _, imp := range request.Imp {
		ext, err := parseExt(imp.Ext)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{Message: err.Error()})
			continue
		}

		requestJSON, err := json.Marshal(buildRequest(request, &imp, ext))
		if err != nil {
			return nil, append(errs, err)
		}

		data := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    requestJSON,
			Headers: headers,
			ImpIDs:  []string{imp.ID},
		}
		result = append(result, data)
	}

	return result, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	err := adapters.CheckResponseStatusCodeForErrors(responseData)
	if err != nil {
		return nil, []error{err}
	}

	respData, price, width, height, errs := parseResponseJson(responseData.Body)
	if len(errs) > 0 {
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

	var bidReqIdRegex = regexp.MustCompile(`^` + request.ID)
	impId := bidReqIdRegex.ReplaceAllLiteralString(respData.ReqID, "")

	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid: &openrtb2.Bid{
			// There's much more possible fields to be added here, see OpenRTB docs for reference (type: Bid)
			ID:      respData.ReqID,
			ImpID:   impId,
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

func buildHeaders(bidReq *openrtb2.BidRequest) http.Header {
	headers := http.Header{}

	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if bidReq.Device != nil && len(bidReq.Device.IP) > 0 {
		headers.Add("X-Forwarded-For", bidReq.Device.IP)
	}

	return headers
}

func buildRequest(bidReq *openrtb2.BidRequest, imp *openrtb2.Imp, ext *openrtb_ext.ImpExtAdQuery) *BidderRequest {
	userId := ""
	if bidReq.User != nil {
		userId = bidReq.User.ID
	}

	bidderRequest := &BidderRequest{
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

	if bidReq.Device != nil {
		bidderRequest.BidIp = bidReq.Device.IP
		bidderRequest.BidIpv6 = bidReq.Device.IPv6
		bidderRequest.BidUa = bidReq.Device.UA
	}

	if bidReq.Site != nil {
		bidderRequest.BidPageUrl = bidReq.Site.Page
	}

	return bidderRequest
}

func parseExt(ext json.RawMessage) (*openrtb_ext.ImpExtAdQuery, error) {
	var bext adapters.ExtImpBidder
	err := jsonutil.Unmarshal(ext, &bext)
	if err != nil {
		return nil, err
	}

	var adsExt openrtb_ext.ImpExtAdQuery
	err = jsonutil.Unmarshal(bext.Bidder, &adsExt)
	if err != nil {
		return nil, err
	}

	// not validating, because it should have been done earlier by the server
	return &adsExt, nil
}

func parseResponseJson(respBody []byte) (*ResponseData, float64, int64, int64, []error) {
	var response ResponseAdQuery
	if err := jsonutil.Unmarshal(respBody, &response); err != nil {
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

	if response.Data.MediaType.Name != openrtb_ext.BidTypeBanner {
		return nil, 0, 0, 0, []error{fmt.Errorf("unsupported MediaType: %s", response.Data.MediaType.Name)}
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
