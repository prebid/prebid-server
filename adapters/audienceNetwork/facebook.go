package audienceNetwork

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type FacebookAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	nonSecureUri string
	platformID   string
	appSecret    string
}

type facebookAdMarkup struct {
	BidID string `json:"bid_id"`
}

var supportedBannerHeights = map[uint64]bool{
	50:  true,
	250: true,
}

// used for cookies and such
func (a *FacebookAdapter) Name() string {
	return "audienceNetwork"
}

func (a *FacebookAdapter) SkipNoCookies() bool {
	return false
}

type facebookReqExt struct {
	PlatformID string `json:"platformid"`
	AuthID     string `json:"authentication_id"`
}

func (this *FacebookAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impressions provided",
		}}
	}

	if request.User == nil || request.User.BuyerUID == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "Missing bidder token in 'user.buyeruid'",
		}}
	}

	return this.buildRequests(request)
}

func (this *FacebookAdapter) buildRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	// Documentation suggests bid request splitting by impression so that each
	// request only represents a single impression
	reqs := make([]*adapters.RequestData, 0, len(request.Imp))
	headers := http.Header{}
	var errs []error

	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Fb-Pool-Routing-Token", request.User.BuyerUID)

	for _, imp := range request.Imp {
		// Make a copy of the request so that we don't change the original request which
		// is shared across multiple threads
		fbreq := *request
		fbreq.Imp = []openrtb.Imp{imp}

		if err := this.modifyRequest(&fbreq); err != nil {
			errs = append(errs, err)
			continue
		}

		body, err := json.Marshal(&fbreq)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		body, err = modifyImpCustom(body, &fbreq.Imp[0])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     this.URI,
			Body:    body,
			Headers: headers,
		})
	}

	return reqs, errs
}

// The authentication ID is a sha256 hmac hash encoded as a hex string, based on
// the app secret and the ID of the bid request
func (this *FacebookAdapter) makeAuthID(req *openrtb.BidRequest) string {
	h := hmac.New(sha256.New, []byte(this.appSecret))
	h.Write([]byte(req.ID))

	return hex.EncodeToString(h.Sum(nil))
}

func (this *FacebookAdapter) modifyRequest(out *openrtb.BidRequest) error {
	if len(out.Imp) != 1 {
		panic("each bid request to facebook should only have a single impression")
	}

	imp := &out.Imp[0]
	plmtId, pubId, err := this.extractPlacementAndPublisher(imp)
	if err != nil {
		return err
	}

	reqExt := facebookReqExt{
		PlatformID: this.platformID,
		AuthID:     this.makeAuthID(out),
	}

	if out.Ext, err = json.Marshal(reqExt); err != nil {
		return err
	}

	imp.TagID = pubId + "_" + plmtId
	imp.Ext = nil

	if out.App != nil {
		app := *out.App
		app.Publisher = &openrtb.Publisher{ID: pubId}
		out.App = &app
	} else {
		site := *out.Site
		site.Publisher = &openrtb.Publisher{ID: pubId}
		out.Site = &site
	}

	if err = this.modifyImp(imp); err != nil {
		return err
	}

	return nil
}

func (this *FacebookAdapter) modifyImp(out *openrtb.Imp) error {
	impType, ok := resolveImpType(out)
	if !ok {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s with invalid type", out.ID),
		}
	}

	if out.Instl == 1 && impType != openrtb_ext.BidTypeBanner {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: interstitial imps are only supported for banner", out.ID),
		}
	}

	switch impType {
	case openrtb_ext.BidTypeBanner:
		if out.Instl == 1 {
			out.Banner.W = openrtb.Uint64Ptr(0)
			out.Banner.H = openrtb.Uint64Ptr(0)
			out.Banner.Format = nil

			return nil
		}

		if out.Banner.H == nil {
			for _, f := range out.Banner.Format {
				if _, ok := supportedBannerHeights[f.H]; ok {
					h := f.H
					out.Banner.H = &h
					break
				}
			}
			if out.Banner.H == nil {
				return &errortypes.BadInput{
					Message: fmt.Sprintf("imp #%s: banner height required", out.ID),
				}
			}
		}

		if _, ok := supportedBannerHeights[*out.Banner.H]; !ok {
			return &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%s: only banner heights 50 and 250 are supported", out.ID),
			}
		}

		/* This will get overwritten post-serialization */
		out.Banner.W = openrtb.Uint64Ptr(0)
		out.Banner.Format = nil
		break
	}

	return nil
}

func (this *FacebookAdapter) extractPlacementAndPublisher(out *openrtb.Imp) (string, string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(out.Ext, &bidderExt); err != nil {
		return "", "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var fbExt openrtb_ext.ExtImpFacebook
	if err := json.Unmarshal(bidderExt.Bidder, &fbExt); err != nil {
		return "", "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	if fbExt.PlacementId == "" {
		return "", "", &errortypes.BadInput{
			Message: "Missing placementId param",
		}
	}

	placementId := fbExt.PlacementId
	publisherId := fbExt.PublisherId

	// Support the legacy path with the caller was expected to pass in just placementId
	// which was an underscore concantenated string with the publisherId and placementId.
	// The new path for callers is to pass in the placementId and publisherId independently
	// and the below code will prefix the placementId that we pass to FAN with the publsiherId
	// so that we can abstract the implementation details from the caller
	toks := strings.Split(placementId, "_")
	if len(toks) == 1 {
		if publisherId == "" {
			return "", "", &errortypes.BadInput{
				Message: "Missing publisherId param",
			}
		}

		return placementId, publisherId, nil
	} else if len(toks) == 2 {
		publisherId = toks[0]
		placementId = toks[1]
	} else {
		return "", "", &errortypes.BadInput{
			Message: fmt.Sprintf("Invalid placementId param '%s' and publisherId param '%s'", placementId, publisherId),
		}
	}

	return placementId, publisherId, nil
}

// XXX: This entire function is just a hack to get around mxmCherry 11.0.0 limitations, without
// having to fork the library and maintain our own branch
func modifyImpCustom(json []byte, imp *openrtb.Imp) ([]byte, error) {
	impType, ok := resolveImpType(imp)
	if ok == false {
		panic("processing an invalid impression")
	}

	var err error

	switch impType {
	case openrtb_ext.BidTypeBanner:
		// The current version of mxmCherry (11.0.0) repesents banner.w as unsigned
		// integers, so setting a value of -1 is not possible which is why we have to do it
		// post-serialization

		// The above does not apply to interstitial impressions
		if imp.Instl == 1 {
			break
		}

		json, err = jsonparser.Set(json, []byte("-1"), "imp", "[0]", "banner", "w")
		if err != nil {
			return json, err
		}

		break

	case openrtb_ext.BidTypeVideo:
		// mxmCherry omits video.w/h if set to zero, so we need to force set those
		// fields to zero post-serialization for the time being
		json, err = jsonparser.Set(json, []byte("0"), "imp", "[0]", "video", "w")
		if err != nil {
			return json, err
		}

		json, err = jsonparser.Set(json, []byte("0"), "imp", "[0]", "video", "h")
		if err != nil {
			return json, err
		}

		break

	case openrtb_ext.BidTypeNative:
		// Set w/h to -1 for native impressions based on the facebook native spec.
		// We have to set this post-serialization since the OpenRTB protocol doesn't
		// actaully support w/h in the native object
		json, err = jsonparser.Set(json, []byte("-1"), "imp", "[0]", "native", "w")
		if err != nil {
			return json, err
		}

		json, err = jsonparser.Set(json, []byte("-1"), "imp", "[0]", "native", "h")
		if err != nil {
			return json, err
		}

		// The FAN adserver does not expect the native request payload, all that information
		// is derived server side based on the placement ID. We need to remove these pieces of
		// information manually since OpenRTB (and thus mxmCherry) never omit native.request
		json = jsonparser.Delete(json, "imp", "[0]", "native", "ver")
		json = jsonparser.Delete(json, "imp", "[0]", "native", "request")

		break
	}

	return json, nil
}

func (this *FacebookAdapter) MakeBids(request *openrtb.BidRequest, adapterRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	/* No bid response */
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	/* Any other http status codes outside of 200 and 204 should be treated as errors */
	if response.StatusCode != http.StatusOK {
		msg := response.Headers.Get("x-fb-an-errors")
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code %d with error message '%s'", response.StatusCode, msg),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	out := adapters.NewBidderResponseWithBidsCapacity(4)
	var errs []error

	for _, seatbid := range bidResp.SeatBid {
		for _, bid := range seatbid.Bid {
			if bid.AdM == "" {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Bid %s missing 'adm'", bid.ID),
				})
				continue
			}

			var obj facebookAdMarkup
			if err := json.Unmarshal([]byte(bid.AdM), &obj); err != nil {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: err.Error(),
				})
				continue
			}

			if obj.BidID == "" {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("bid %s missing 'bid_id' in 'adm'", bid.ID),
				})
				continue
			}

			bid.AdID = obj.BidID
			bid.CrID = obj.BidID

			out.Bids = append(out.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: resolveBidType(&bid, request),
			})
		}
	}

	return out, errs
}

func resolveBidType(bid *openrtb.Bid, req *openrtb.BidRequest) openrtb_ext.BidType {
	for _, imp := range req.Imp {
		if bid.ImpID == imp.ID {
			if typ, ok := resolveImpType(&imp); ok {
				return typ
			}

			panic("Processing an invalid impression; cannot resolve impression type")
		}
	}

	panic(fmt.Sprintf("Invalid bid imp ID %s does not match any imp IDs from the original bid request", bid.ImpID))
}

func resolveImpType(imp *openrtb.Imp) (openrtb_ext.BidType, bool) {
	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner, true
	}

	if imp.Video != nil {
		return openrtb_ext.BidTypeVideo, true
	}

	if imp.Audio != nil {
		return openrtb_ext.BidTypeAudio, true
	}

	if imp.Native != nil {
		return openrtb_ext.BidTypeNative, true
	}

	return openrtb_ext.BidTypeBanner, false
}

func NewFacebookBidder(client *http.Client, platformID string, appSecret string) adapters.Bidder {
	if platformID == "" {
		glog.Errorf("No facebook partnerID specified. Calls to the Audience Network will fail. Did you set adapters.facebook.platform_id in the app config?")
		return &adapters.MisconfiguredBidder{
			Name:  "audienceNetwork",
			Error: errors.New("Audience Network is not configured properly on this Prebid Server deploy. If you believe this should work, contact the company hosting the service and tell them to check their configuration."),
		}
	}

	if appSecret == "" {
		glog.Errorf("No facebook app secret specified. Calls to the Audience Network will fail. Did you set adapters.facebook.app_secret in the app config?")
		return &adapters.MisconfiguredBidder{
			Name:  "audienceNetwork",
			Error: errors.New("Audience Network is not configured properly on this Prebid Server deploy. If you believe this should work, contact the company hosting the service and tell them to check their configuration."),
		}
	}

	a := &adapters.HTTPAdapter{Client: client}

	return &FacebookAdapter{
		http: a,
		URI:  "https://an.facebook.com/placementbid.ortb",
		//for AB test
		nonSecureUri: "http://an.facebook.com/placementbid.ortb",
		platformID:   platformID,
		appSecret:    appSecret,
	}
}

func (fa *FacebookAdapter) MakeTimeoutNotification(req *adapters.RequestData) (*adapters.RequestData, []error) {
	// Note, facebook creates one request per imp, so all these requests will only have one imp in them
	auction_id, err := jsonparser.GetString(req.Body, "imp", "[0]", "id")
	if err != nil {
		return &adapters.RequestData{}, []error{err}
	}

	uri := fmt.Sprintf("https://www.facebook.com/audiencenetwork/nurl/?partner=%s&app=%s&auction=%s&ortb_loss_code=2", fa.platformID, fa.platformID, auction_id)
	timeoutReq := adapters.RequestData{
		Method:  "GET",
		Uri:     uri,
		Body:    nil,
		Headers: http.Header{},
	}
	return &timeoutReq, nil
}
