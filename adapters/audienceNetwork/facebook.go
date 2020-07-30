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

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/maputil"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
)

type FacebookAdapter struct {
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

	if request.Site != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "Site impressions are not supported.",
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

	// Every outgoing FAN request has a single impression, so we can safely use the unique
	// impression ID as the FAN request ID. We need to make sure that we update the request
	// ID *BEFORE* we generate the auth ID since its a hash based on the request ID
	out.ID = imp.ID

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

	if impType == openrtb_ext.BidTypeBanner {
		bannerCopy := *out.Banner
		out.Banner = &bannerCopy

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

	placementID := fbExt.PlacementId
	publisherID := fbExt.PublisherId

	// Support the legacy path with the caller was expected to pass in just placementId
	// which was an underscore concantenated string with the publisherId and placementId.
	// The new path for callers is to pass in the placementId and publisherId independently
	// and the below code will prefix the placementId that we pass to FAN with the publsiherId
	// so that we can abstract the implementation details from the caller
	toks := strings.Split(placementID, "_")
	if len(toks) == 1 {
		if publisherID == "" {
			return "", "", &errortypes.BadInput{
				Message: "Missing publisherId param",
			}
		}

		return placementID, publisherID, nil
	} else if len(toks) == 2 {
		publisherID = toks[0]
		placementID = toks[1]
	} else {
		return "", "", &errortypes.BadInput{
			Message: fmt.Sprintf("Invalid placementId param '%s' and publisherId param '%s'", placementID, publisherID),
		}
	}

	return placementID, publisherID, nil
}

// XXX: This entire function is just a hack to get around mxmCherry 11.0.0 limitations, without
// having to fork the library and maintain our own branch
func modifyImpCustom(jsonData []byte, imp *openrtb.Imp) ([]byte, error) {
	impType, ok := resolveImpType(imp)
	if ok == false {
		panic("processing an invalid impression")
	}

	var jsonMap map[string]interface{}
	err := json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		return jsonData, err
	}

	var impMap map[string]interface{}
	if impSlice, ok := maputil.ReadEmbeddedSlice(jsonMap, "imp"); !ok {
		return jsonData, errors.New("unable to find imp in json data")
	} else if len(impSlice) == 0 {
		return jsonData, errors.New("unable to find imp[0] in json data")
	} else if impMap, ok = impSlice[0].(map[string]interface{}); !ok {
		return jsonData, errors.New("unexpected type for imp[0] found in json data")
	}

	switch impType {
	case openrtb_ext.BidTypeBanner:
		// The current version of mxmCherry (11.0.0) represents banner.w as an unsigned
		// integer, so setting a value of -1 is not possible which is why we have to do it
		// post-serialization
		isInterstitial := imp.Instl == 1
		if !isInterstitial {
			if bannerMap, ok := maputil.ReadEmbeddedMap(impMap, "banner"); ok {
				bannerMap["w"] = json.RawMessage("-1")
			} else {
				return jsonData, errors.New("unable to find imp[0].banner in json data")
			}
		}

	case openrtb_ext.BidTypeVideo:
		// mxmCherry omits video.w/h if set to zero, so we need to force set those
		// fields to zero post-serialization for the time being
		if videoMap, ok := maputil.ReadEmbeddedMap(impMap, "video"); ok {
			videoMap["w"] = json.RawMessage("0")
			videoMap["h"] = json.RawMessage("0")
		} else {
			return jsonData, errors.New("unable to find imp[0].video in json data")
		}

	case openrtb_ext.BidTypeNative:
		nativeMap, ok := maputil.ReadEmbeddedMap(impMap, "native")
		if !ok {
			return jsonData, errors.New("unable to find imp[0].video in json data")
		}

		// Set w/h to -1 for native impressions based on the facebook native spec.
		// We have to set this post-serialization since the OpenRTB protocol doesn't
		// actually support w/h in the native object
		nativeMap["w"] = json.RawMessage("-1")
		nativeMap["h"] = json.RawMessage("-1")

		// The FAN adserver does not expect the native request payload, all that information
		// is derived server side based on the placement ID. We need to remove these pieces of
		// information manually since OpenRTB (and thus mxmCherry) never omit native.request
		delete(nativeMap, "ver")
		delete(nativeMap, "request")
	}

	if jsonReEncoded, err := json.Marshal(jsonMap); err == nil {
		return jsonReEncoded, nil
	} else {
		return nil, fmt.Errorf("unable to encode json data (%v)", err)
	}
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

func NewFacebookBidder(platformID string, appSecret string) adapters.Bidder {
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

	return &FacebookAdapter{
		URI: "https://an.facebook.com/placementbid.ortb",
		//for AB test
		nonSecureUri: "http://an.facebook.com/placementbid.ortb",
		platformID:   platformID,
		appSecret:    appSecret,
	}
}

func (fa *FacebookAdapter) MakeTimeoutNotification(req *adapters.RequestData) (*adapters.RequestData, []error) {
	var (
		rID   string
		pubID string
		err   error
	)

	// Note, the facebook adserver can only handle single impression requests, so we have to split multi-imp requests into
	// multiple request. In order to ensure that every split request has a unique ID, the split request IDs are set to the
	// corresponding imp's ID
	rID, err = jsonparser.GetString(req.Body, "id")
	if err != nil {
		return &adapters.RequestData{}, []error{err}
	}

	// The publisher ID is expected in the app object
	pubID, err = jsonparser.GetString(req.Body, "app", "publisher", "id")
	if err != nil {
		return &adapters.RequestData{}, []error{
			errors.New("path app.publisher.id not found in the request"),
		}
	}

	uri := fmt.Sprintf("https://www.facebook.com/audiencenetwork/nurl/?partner=%s&app=%s&auction=%s&ortb_loss_code=2", fa.platformID, pubID, rID)
	timeoutReq := adapters.RequestData{
		Method:  "GET",
		Uri:     uri,
		Body:    nil,
		Headers: http.Header{},
	}

	return &timeoutReq, nil
}
