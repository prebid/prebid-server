package appnexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const uri = "http://ib.adnxs.com/openrtb2"

type AppNexusAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *AppNexusAdapter) Name() string {
	return "AppNexus"
}

// used for cookies and such
func (a *AppNexusAdapter) FamilyName() string {
	return "adnxs"
}

func (a *AppNexusAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *AppNexusAdapter) SkipNoCookies() bool {
	return false
}

type KeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

type appnexusParams struct {
	PlacementId       int      `json:"placementId"`
	InvCode           string   `json:"invCode"`
	Member            string   `json:"member"`
	Keywords          []KeyVal `json:"keywords"`
	TrafficSourceCode string   `json:"trafficSourceCode"`
	Reserve           float64  `json:"reserve"`
	Position          string   `json:"position"`
}

type appnexusImpExtAppnexus struct {
	PlacementID       int    `json:"placement_id,omitempty"`
	Keywords          string `json:"keywords,omitempty"`
	TrafficSourceCode string `json:"traffic_source_code,omitempty"`
}

type appnexusImpExt struct {
	Appnexus appnexusImpExtAppnexus `json:"appnexus"`
}

func (a *AppNexusAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	anReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.FamilyName(), supportedMediaTypes, true)

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

		if params.PlacementId == 0 && (params.InvCode == "" || params.Member == "") {
			return nil, errors.New("No placement or member+invcode provided")
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

	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			numBids++

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

			mediaType := getMediaTypeForImp(bid.ImpID, anReq.Imp)
			pbid.CreativeMediaType = string(mediaType)
			bids = append(bids, &pbid)
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
	// We only support banner and video impressions for now.
	if imp.Native != nil || imp.Audio != nil {
		return "", fmt.Errorf("Appnexus doesn't support audio or native Imps. Ignoring Imp ID=%s", imp.ID)
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", err
	}

	var appnexusExt openrtb_ext.ExtImpAppnexus
	if err := json.Unmarshal(bidderExt.Bidder, &appnexusExt); err != nil {
		return "", err
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
	if imp.Banner != nil && appnexusExt.Position != "" {
		if appnexusExt.Position == "above" {
			imp.Banner.Pos = openrtb.AdPositionAboveTheFold.Ptr()
		} else if appnexusExt.Position == "below" {
			imp.Banner.Pos = openrtb.AdPositionBelowTheFold.Ptr()
		}
	}

	impExt := appnexusImpExt{Appnexus: appnexusImpExtAppnexus{
		PlacementID:       appnexusExt.PlacementId,
		TrafficSourceCode: appnexusExt.TrafficSourceCode,
		Keywords:          makeKeywordStr(appnexusExt.Keywords),
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

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bids = append(bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
			})
		}
	}
	return bids, nil
}

// getMediaTypeForImp figures out which media type this bid is for.
//
// This is only safe for multi-type impressions because the AN server prioritizes video over banner,
// and we duplicate that logic here. A ticket exists to return the media type in the bid response,
// at which point we can delete this.
func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}

func NewAppNexusAdapter(config *adapters.HTTPAdapterConfig, externalURL string) *AppNexusAdapter {
	return NewAppNexusBidder(adapters.NewHTTPAdapter(config).Client, externalURL)
}

func NewAppNexusBidder(client *http.Client, externalURL string) *AppNexusAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &AppNexusAdapter{
		http:         a,
		URI:          uri,
		usersyncInfo: info,
	}
}
