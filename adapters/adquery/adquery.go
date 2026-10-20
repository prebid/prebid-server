package adquery

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	defaultCurrency string = "PLN"
	bidderName      string = "adquery"
	prebidVersion   string = "server"
	eidSource       string = "adquery.io"
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

	// The EIDs are the same for every imp, so they are computed once per request.
	eids, err := getEids(request.User)
	if err != nil {
		errs = append(errs, err)
	}

	for _, imp := range request.Imp {
		ext, err := parseExt(imp.Ext)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{Message: err.Error()})
			continue
		}

		bidderRequest := buildRequest(request, &imp, ext)
		bidderRequest.Eids = eids

		requestJSON, err := json.Marshal(bidderRequest)
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

// maxEidUids caps the number of uids forwarded in Eids, across all EIDs.
const maxEidUids = 50

// getEids returns the user's EIDs to forward to adQuery with their original source. EIDs without a
// source and uids without an id are dropped, and identical duplicates are removed. The order does
// not depend on input order: adquery.io first, then by source and uids, ties broken by the remaining
// fields (ext is compared as raw JSON). The cap is applied after sorting, so it never drops the
// adquery.io uid. The request is not modified.
func getEids(user *openrtb2.User) ([]openrtb2.EID, error) {
	source, err := userEids(user)
	if err != nil {
		return nil, err
	}

	cleaned := make([]openrtb2.EID, 0, len(source))
	for _, eid := range source {
		if cleanedEid, ok := cleanEid(eid); ok {
			cleaned = append(cleaned, cleanedEid)
		}
	}
	cleaned = sortUniqueEids(cleaned)

	var result []openrtb2.EID
	budget := maxEidUids
	for _, eid := range cleaned {
		if budget <= 0 {
			break
		}
		if len(eid.UIDs) > budget {
			eid.UIDs = eid.UIDs[:budget]
		}
		result = append(result, eid)
		budget -= len(eid.UIDs)
	}
	// Cutting uids can turn an EID into a copy of another one, so the capped list is sorted and
	// deduplicated again; adquery.io still sorts first.
	return sortUniqueEids(result), nil
}

func sortUniqueEids(eids []openrtb2.EID) []openrtb2.EID {
	slices.SortFunc(eids, compareEids)
	return slices.CompactFunc(eids, func(a, b openrtb2.EID) bool { return compareEids(a, b) == 0 })
}

// userEids returns user.eids (ORTB 2.6) when set and user.ext.eids (2.5) otherwise, like PBS core.
// adquery declares no openrtb version in its bidder info, so PBS downconverts to 2.5 and the EIDs
// arrive in user.ext.eids; user.eids only matters if the bidder info ever declares 2.6. Only the
// eids field of user.ext is decoded, so a malformed unrelated field does not drop the EIDs; an
// undecodable user.ext.eids is skipped with a warning.
func userEids(user *openrtb2.User) ([]openrtb2.EID, error) {
	if user == nil {
		return nil, nil
	}
	if len(user.EIDs) > 0 || len(user.Ext) == 0 {
		return user.EIDs, nil
	}

	var userExt struct {
		Eids []openrtb2.EID `json:"eids"`
	}
	if err := jsonutil.Unmarshal(user.Ext, &userExt); err != nil {
		return nil, &errortypes.Warning{Message: fmt.Sprintf("ignoring user.ext.eids: %s", err.Error())}
	}
	return userExt.Eids, nil
}

// cleanEid returns a copy of eid with only its uids that have an id, sorted and without identical
// duplicates; ok is false when eid has no source or no such uid. ext values are compacted, so the
// same JSON with different whitespace compares (and dedupes) as equal. The input is not modified.
func cleanEid(eid openrtb2.EID) (openrtb2.EID, bool) {
	if eid.Source == "" {
		return eid, false
	}
	uids := make([]openrtb2.UID, 0, len(eid.UIDs))
	for _, uid := range eid.UIDs {
		if uid.ID != "" {
			uid.Ext = compactJSON(uid.Ext)
			uids = append(uids, uid)
		}
	}
	if len(uids) == 0 {
		return eid, false
	}
	slices.SortFunc(uids, compareUids)
	eid.UIDs = slices.CompactFunc(uids, func(a, b openrtb2.UID) bool { return compareUids(a, b) == 0 })
	eid.Ext = compactJSON(eid.Ext)
	return eid, true
}

// compactJSON returns raw without insignificant whitespace, or raw itself when it is empty or invalid.
func compactJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return raw
	}
	return buf.Bytes()
}

func compareUids(a, b openrtb2.UID) int {
	return cmp.Or(
		strings.Compare(a.ID, b.ID),
		cmp.Compare(a.AType, b.AType),
		bytes.Compare(a.Ext, b.Ext),
	)
}

func compareEids(a, b openrtb2.EID) int {
	aIsAdquery := a.Source == eidSource
	bIsAdquery := b.Source == eidSource
	if aIsAdquery != bIsAdquery {
		if aIsAdquery {
			return -1
		}
		return 1
	}
	return cmp.Or(
		strings.Compare(a.Source, b.Source),
		slices.CompareFunc(a.UIDs, b.UIDs, compareUids),
		strings.Compare(a.Inserter, b.Inserter),
		strings.Compare(a.Matcher, b.Matcher),
		cmp.Compare(a.MM, b.MM),
		bytes.Compare(a.Ext, b.Ext),
	)
}
