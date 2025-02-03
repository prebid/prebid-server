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
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/maputil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

var supportedBannerHeights = map[int64]struct{}{
	50:  {},
	250: {},
}

type adapter struct {
	uri        string
	platformID string
	appSecret  string
}

type facebookAdMarkup struct {
	BidID string `json:"bid_id"`
}

type facebookReqExt struct {
	PlatformID string `json:"platformid"`
	AuthID     string `json:"authentication_id"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

	return a.buildRequests(request)
}

func (a *adapter) buildRequests(request *openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
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
		fbreq.Imp = []openrtb2.Imp{imp}

		if err := a.modifyRequest(&fbreq); err != nil {
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

		body, err = jsonutil.DropElement(body, "consented_providers_settings")
		if err != nil {
			errs = append(errs, err)
			return reqs, errs
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.uri,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(fbreq.Imp),
		})
	}

	return reqs, errs
}

// The authentication ID is a sha256 hmac hash encoded as a hex string, based on
// the app secret and the ID of the bid request
func (a *adapter) makeAuthID(req *openrtb2.BidRequest) string {
	h := hmac.New(sha256.New, []byte(a.appSecret))
	h.Write([]byte(req.ID))

	return hex.EncodeToString(h.Sum(nil))
}

func (a *adapter) modifyRequest(out *openrtb2.BidRequest) error {
	if len(out.Imp) != 1 {
		panic("each bid request to facebook should only have a single impression")
	}

	imp := &out.Imp[0]
	plmtId, pubId, err := extractPlacementAndPublisher(imp)
	if err != nil {
		return err
	}

	// Every outgoing FAN request has a single impression, so we can safely use the unique
	// impression ID as the FAN request ID. We need to make sure that we update the request
	// ID *BEFORE* we generate the auth ID since its a hash based on the request ID
	out.ID = imp.ID

	reqExt := facebookReqExt{
		PlatformID: a.platformID,
		AuthID:     a.makeAuthID(out),
	}

	if out.Ext, err = json.Marshal(reqExt); err != nil {
		return err
	}

	imp.TagID = pubId + "_" + plmtId
	imp.Ext = nil

	if out.App != nil {
		app := *out.App
		app.Publisher = &openrtb2.Publisher{ID: pubId}
		out.App = &app
	}

	if err = modifyImp(imp); err != nil {
		return err
	}

	return nil
}

func modifyImp(out *openrtb2.Imp) error {
	impType := resolveImpType(out)

	if out.Instl == 1 && impType != openrtb_ext.BidTypeBanner {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp #%s: interstitial imps are only supported for banner", out.ID),
		}
	}

	if impType == openrtb_ext.BidTypeBanner {
		bannerCopy := *out.Banner
		out.Banner = &bannerCopy

		if out.Instl == 1 {
			out.Banner.W = ptrutil.ToPtr[int64](0)
			out.Banner.H = ptrutil.ToPtr[int64](0)
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

		out.Banner.W = ptrutil.ToPtr[int64](-1)
		out.Banner.Format = nil
	}

	return nil
}

func extractPlacementAndPublisher(out *openrtb2.Imp) (string, string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(out.Ext, &bidderExt); err != nil {
		return "", "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var fbExt openrtb_ext.ExtImpFacebook
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &fbExt); err != nil {
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
	// which was an underscore concatenated string with the publisherId and placementId.
	// The new path for callers is to pass in the placementId and publisherId independently
	// and the below code will prefix the placementId that we pass to FAN with the publisherId
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

// modifyImpCustom modifies the impression after it's marshalled to add a non-openrtb field.
func modifyImpCustom(jsonData []byte, imp *openrtb2.Imp) ([]byte, error) {
	impType := resolveImpType(imp)

	// we only need to modify video and native impressions
	if impType != openrtb_ext.BidTypeVideo && impType != openrtb_ext.BidTypeNative {
		return jsonData, nil
	}

	var jsonMap map[string]interface{}
	if err := jsonutil.Unmarshal(jsonData, &jsonMap); err != nil {
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
	case openrtb_ext.BidTypeVideo:
		videoMap, ok := maputil.ReadEmbeddedMap(impMap, "video")
		if !ok {
			return jsonData, errors.New("unable to find imp[0].video in json data")
		}

		// the openrtb library omits video.w/h if set to zero, so we need to force set those
		// fields to zero post-serialization for the time being
		videoMap["w"] = json.RawMessage("0")
		videoMap["h"] = json.RawMessage("0")

	case openrtb_ext.BidTypeNative:
		nativeMap, ok := maputil.ReadEmbeddedMap(impMap, "native")
		if !ok {
			return jsonData, errors.New("unable to find imp[0].video in json data")
		}

		// Set w/h to -1 for native impressions based on the facebook native spec.
		// We have to set this post-serialization since these fields are not included
		// in the OpenRTB 2.5 spec.
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

func (a *adapter) MakeBids(request *openrtb2.BidRequest, adapterRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		msg := response.Headers.Get("x-fb-an-errors")
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code %d with error message '%s'", response.StatusCode, msg),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	out := adapters.NewBidderResponseWithBidsCapacity(4)
	var errs []error

	for _, seatbid := range bidResp.SeatBid {
		for i := range seatbid.Bid {
			bid := seatbid.Bid[i]

			if bid.AdM == "" {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Bid %s missing 'adm'", bid.ID),
				})
				continue
			}

			var obj facebookAdMarkup
			if err := jsonutil.Unmarshal([]byte(bid.AdM), &obj); err != nil {
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

func resolveBidType(bid *openrtb2.Bid, req *openrtb2.BidRequest) openrtb_ext.BidType {
	for _, imp := range req.Imp {
		if bid.ImpID == imp.ID {
			return resolveImpType(&imp)
		}
	}

	panic(fmt.Sprintf("Invalid bid imp ID %s does not match any imp IDs from the original bid request", bid.ImpID))
}

func resolveImpType(imp *openrtb2.Imp) openrtb_ext.BidType {
	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner
	}

	if imp.Video != nil {
		return openrtb_ext.BidTypeVideo
	}

	if imp.Audio != nil {
		return openrtb_ext.BidTypeAudio
	}

	if imp.Native != nil {
		return openrtb_ext.BidTypeNative
	}

	// Required to satisfy compiler. Not reachable in practice due to validations performed in PBS-Core.
	return openrtb_ext.BidTypeBanner
}

// Builder builds a new instance of Facebook's Audience Network adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	if config.PlatformID == "" {
		return nil, errors.New("PartnerID is not configured. Did you set adapters.facebook.platform_id in the app config?")
	}

	if config.AppSecret == "" {
		return nil, errors.New("AppSecret is not configured. Did you set adapters.facebook.app_secret in the app config?")
	}

	bidder := &adapter{
		uri:        config.Endpoint,
		platformID: config.PlatformID,
		appSecret:  config.AppSecret,
	}
	return bidder, nil
}

func (a *adapter) MakeTimeoutNotification(req *adapters.RequestData) (*adapters.RequestData, []error) {
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

	uri := fmt.Sprintf("https://www.facebook.com/audiencenetwork/nurl/?partner=%s&app=%s&auction=%s&ortb_loss_code=2", a.platformID, pubID, rID)
	timeoutReq := adapters.RequestData{
		Method:  "GET",
		Uri:     uri,
		Body:    nil,
		Headers: http.Header{},
	}

	return &timeoutReq, nil
}
