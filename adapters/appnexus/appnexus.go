package appnexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Docs for this API can be found at https://wiki.appnexus.com/display/supply/Incoming+Bid+Request+from+SSPs
const uri = "http://ib.adnxs.com/openrtb2"

type AppNexusAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *AppNexusAdapter) Name() string {
	return "adnxs"
}

func (a *AppNexusAdapter) SkipNoCookies() bool {
	return false
}

type KeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

type appnexusParams struct {
	LegacyPlacementId       int             `json:"placementId"`
	LegacyInvCode           string          `json:"invCode"`
	LegacyTrafficSourceCode string          `json:"trafficSourceCode"`
	PlacementId             int             `json:"placement_id"`
	InvCode                 string          `json:"inv_code"`
	Member                  string          `json:"member"`
	Keywords                []KeyVal        `json:"keywords"`
	TrafficSourceCode       string          `json:"traffic_source_code"`
	Reserve                 float64         `json:"reserve"`
	Position                string          `json:"position"`
	UsePmtRule              *bool           `json:"use_pmt_rule"`
	PrivateSizes            json.RawMessage `json:"private_sizes"`
}

type appnexusImpExtAppnexus struct {
	PlacementID       int             `json:"placement_id,omitempty"`
	Keywords          string          `json:"keywords,omitempty"`
	TrafficSourceCode string          `json:"traffic_source_code,omitempty"`
	UsePmtRule        *bool           `json:"use_pmt_rule,omitempty"`
	PrivateSizes      json.RawMessage `json:"private_sizes,omitempty"`
}

type appnexusBidExt struct {
	Appnexus appnexusBidExtAppnexus `json:"appnexus"`
}

type appnexusBidExtAppnexus struct {
	BidType int `json:"bid_ad_type"`
}

type appnexusImpExt struct {
	Appnexus appnexusImpExtAppnexus `json:"appnexus"`
}

func (a *AppNexusAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	anReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), supportedMediaTypes, true)

	if err != nil {
		return nil, err
	}
	uri := a.URI
	for i, unit := range bidder.AdUnits {
		var params appnexusParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		// Accept legacy Appnexus parameters if we don't have modern ones
		// Don't worry if both is set as validation rules should prevent, and this is temporary anyway.
		if params.PlacementId == 0 && params.LegacyPlacementId != 0 {
			params.PlacementId = params.LegacyPlacementId
		}
		if params.InvCode == "" && params.LegacyInvCode != "" {
			params.InvCode = params.LegacyInvCode
		}
		if params.TrafficSourceCode == "" && params.LegacyTrafficSourceCode != "" {
			params.TrafficSourceCode = params.LegacyTrafficSourceCode
		}

		if params.PlacementId == 0 && (params.InvCode == "" || params.Member == "") {
			return nil, errors.New("No placement or member+invcode provided")
		}

		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(anReq.Imp) <= i {
			break
		}
		if params.InvCode != "" {
			anReq.Imp[i].TagID = params.InvCode
			if params.Member != "" {
				// this assumes that the same member ID is used across all tags, which should be the case
				uri = fmt.Sprintf("%s?member_id=%s", a.URI, params.Member)
			}

		}
		if params.Reserve > 0 {
			anReq.Imp[i].BidFloor = params.Reserve // TODO: we need to factor in currency here if non-USD
		}
		if anReq.Imp[i].Banner != nil && params.Position != "" {
			if params.Position == "above" {
				anReq.Imp[i].Banner.Pos = openrtb.AdPositionAboveTheFold.Ptr()
			} else if params.Position == "below" {
				anReq.Imp[i].Banner.Pos = openrtb.AdPositionBelowTheFold.Ptr()
			}
		}

		kvs := make([]string, 0, len(params.Keywords)*2)
		for _, kv := range params.Keywords {
			if len(kv.Values) == 0 {
				kvs = append(kvs, kv.Key)
			} else {
				for _, val := range kv.Values {
					kvs = append(kvs, fmt.Sprintf("%s=%s", kv.Key, val))
				}

			}
		}

		keywordStr := strings.Join(kvs, ",")

		impExt := appnexusImpExt{Appnexus: appnexusImpExtAppnexus{
			PlacementID:       params.PlacementId,
			TrafficSourceCode: params.TrafficSourceCode,
			Keywords:          keywordStr,
			UsePmtRule:        params.UsePmtRule,
			PrivateSizes:      params.PrivateSizes,
		}}
		anReq.Imp[i].Ext, err = json.Marshal(&impExt)
	}

	reqJSON, err := json.Marshal(anReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: uri,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = anResp.StatusCode

	if anResp.StatusCode == 204 {
		return nil, nil
	}

	defer anResp.Body.Close()
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return nil, err
	}
	responseBody := string(body)

	if anResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status %d; body: %s", anResp.StatusCode, responseBody)
	}

	if req.IsDebug {
		debug.ResponseBody = responseBody
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, err
	}

	bids := make(pbs.PBSBidSlice, 0)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, fmt.Errorf("Unknown ad unit code '%s'", bid.ImpID)
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
				NURL:        bid.NURL,
			}

			if mediaType, err := getMediaTypeForBid(&bid); err == nil {
				pbid.CreativeMediaType = string(mediaType)
				bids = append(bids, &pbid)
			}
		}
	}

	return bids, nil
}

func (a *AppNexusAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	memberIds := make(map[string]bool)
	errs := make([]error, 0, len(request.Imp))

	for i := 0; i < len(request.Imp); i++ {
		memberId, err := preprocess(&request.Imp[i])
		if memberId != "" {
			memberIds[memberId] = true
		}
		// If the preprocessing failed, the server won't be able to bid on this Imp. Delete it, and note the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}

	thisUri := uri

	// The Appnexus API requires a Member ID in the URL. This means the request may fail if
	// different impressions have different member IDs.
	// Check for this condition, and log an error if it's a problem.
	if len(memberIds) > 0 {
		uniqueIds := keys(memberIds)
		memberId := uniqueIds[0]
		thisUri = fmt.Sprintf("%s?member_id=%s", thisUri, memberId)

		if len(uniqueIds) > 1 {
			errs = append(errs, fmt.Errorf("All request.imp[i].ext.appnexus.member params must match. Request contained: %v", uniqueIds))
		}
	}

	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisUri,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

// get the keys from the map
func keys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key, _ := range m {
		keys = append(keys, key)
	}
	return keys
}

// preprocess mutates the imp to get it ready to send to appnexus.
//
// It returns the member param, if it exists, and an error if anything went wrong during the preprocessing.
func preprocess(imp *openrtb.Imp) (string, error) {
	// We don't support audio imps yet.
	if imp.Audio != nil {
		return "", fmt.Errorf("Appnexus doesn't support audio Imps. Ignoring Imp ID=%s", imp.ID)
	}
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", err
	}

	var appnexusExt openrtb_ext.ExtImpAppnexus
	if err := json.Unmarshal(bidderExt.Bidder, &appnexusExt); err != nil {
		return "", err
	}

	// Accept legacy Appnexus parameters if we don't have modern ones
	// Don't worry if both is set as validation rules should prevent, and this is temporary anyway.
	if appnexusExt.PlacementId == 0 && appnexusExt.LegacyPlacementId != 0 {
		appnexusExt.PlacementId = appnexusExt.LegacyPlacementId
	}
	if appnexusExt.InvCode == "" && appnexusExt.LegacyInvCode != "" {
		appnexusExt.InvCode = appnexusExt.LegacyInvCode
	}
	if appnexusExt.TrafficSourceCode == "" && appnexusExt.LegacyTrafficSourceCode != "" {
		appnexusExt.TrafficSourceCode = appnexusExt.LegacyTrafficSourceCode
	}

	if appnexusExt.PlacementId == 0 && (appnexusExt.InvCode == "" || appnexusExt.Member == "") {
		return "", errors.New("No placement or member+invcode provided")
	}

	if appnexusExt.InvCode != "" {
		imp.TagID = appnexusExt.InvCode
	}
	if appnexusExt.Reserve > 0 {
		imp.BidFloor = appnexusExt.Reserve // This will be broken for non-USD currency.
	}
	if imp.Banner != nil {
		bannerCopy := *imp.Banner
		if appnexusExt.Position == "above" {
			bannerCopy.Pos = openrtb.AdPositionAboveTheFold.Ptr()
		} else if appnexusExt.Position == "below" {
			bannerCopy.Pos = openrtb.AdPositionBelowTheFold.Ptr()
		}

		// Fixes #307
		if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
			firstFormat := bannerCopy.Format[0]
			bannerCopy.W = &(firstFormat.W)
			bannerCopy.H = &(firstFormat.H)
		}
		imp.Banner = &bannerCopy
	}

	impExt := appnexusImpExt{Appnexus: appnexusImpExtAppnexus{
		PlacementID:       appnexusExt.PlacementId,
		TrafficSourceCode: appnexusExt.TrafficSourceCode,
		Keywords:          makeKeywordStr(appnexusExt.Keywords),
		UsePmtRule:        appnexusExt.UsePmtRule,
		PrivateSizes:      appnexusExt.PrivateSizes,
	}}
	var err error
	if imp.Ext, err = json.Marshal(&impExt); err != nil {
		return appnexusExt.Member, err
	}

	return appnexusExt.Member, nil
}

func makeKeywordStr(keywords []*openrtb_ext.ExtImpAppnexusKeyVal) string {
	kvs := make([]string, 0, len(keywords)*2)
	for _, kv := range keywords {
		if len(kv.Values) == 0 {
			kvs = append(kvs, kv.Key)
		} else {
			for _, val := range kv.Values {
				kvs = append(kvs, fmt.Sprintf("%s=%s", kv.Key, val))
			}
		}
	}

	return strings.Join(kvs, ",")
}

func (a *AppNexusAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) ([]*adapters.TypedBid, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bids := make([]*adapters.TypedBid, 0, 5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			if bidType, err := getMediaTypeForBid(&bid); err == nil {
				bids = append(bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			} else {
				errs = append(errs, err)
			}
		}
	}
	return bids, errs
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *openrtb.Bid) (openrtb_ext.BidType, error) {
	var impExt appnexusBidExt
	if err := json.Unmarshal(bid.Ext, &impExt); err != nil {
		return "", err
	}
	switch impExt.Appnexus.BidType {
	case 0:
		return openrtb_ext.BidTypeBanner, nil
	case 1:
		return openrtb_ext.BidTypeVideo, nil
	case 2:
		return openrtb_ext.BidTypeAudio, nil
	case 3:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unrecognized bid_ad_type in response from appnexus: %d", impExt.Appnexus.BidType)
	}
}

func NewAppNexusAdapter(config *adapters.HTTPAdapterConfig) *AppNexusAdapter {
	return NewAppNexusBidder(adapters.NewHTTPAdapter(config).Client)
}

func NewAppNexusBidder(client *http.Client) *AppNexusAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &AppNexusAdapter{
		http: a,
		URI:  uri,
	}
}
