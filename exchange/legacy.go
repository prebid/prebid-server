package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

// AdaptLegacyAdapter turns a bidder.Adapter into an adaptedBidder.
//
// This is a temporary function which helps make the transition to OpenRTB smooth. Bidders which have not been
// updated yet can use this to be "OpenRTB-ish". They'll bid as well as they can, given the limitations of the
// legacy protocol
func adaptLegacyAdapter(adapter adapters.Adapter) adaptedBidder {
	return &adaptedAdapter{
		adapter: adapter,
	}
}

type adaptedAdapter struct {
	adapter adapters.Adapter
}

// requestBid attempts to bid on OpenRTB requests using the legacy protocol.
//
// This is not ideal. OpenRTB provides a superset of the legacy data structures.
// For requests which use those features, the best we can do is respond with "no bid".
func (bidder *adaptedAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest, bidderTarg *targetData, name openrtb_ext.BidderName, to *analytics.AuctionObject) (*pbsOrtbSeatBid, []error) {
	legacyRequest, legacyBidder, errs := bidder.toLegacyAdapterInputs(request, name)
	if legacyRequest == nil || legacyBidder == nil {
		return nil, errs
	}

	legacyBids, err := bidder.adapter.Call(ctx, legacyRequest, legacyBidder)
	if err != nil {
		errs = append(errs, err)
	}

	finalResponse, moreErrs := toNewResponse(legacyBids, legacyBidder, bidderTarg, name)
	return finalResponse, append(errs, moreErrs...)
}

// ----------------------------------------------------------------------------
// Request transformations.

// toLegacyAdapterInputs is a best-effort transformation of an OpenRTB BidRequest into the args needed to run a legacy Adapter.
// If the OpenRTB request is too complex, it fails with an error.
// If the error is nil, then the PBSRequest and PBSBidder are valid.
func (bidder *adaptedAdapter) toLegacyAdapterInputs(req *openrtb.BidRequest, name openrtb_ext.BidderName) (*pbs.PBSRequest, *pbs.PBSBidder, []error) {
	legacyReq, err := bidder.toLegacyRequest(req)
	if err != nil {
		return nil, nil, []error{err}
	}

	legacyBidder, errs := toLegacyBidder(req, name)
	if legacyBidder == nil {
		return nil, nil, errs
	}

	return legacyReq, legacyBidder, errs
}

func (bidder *adaptedAdapter) toLegacyRequest(req *openrtb.BidRequest) (*pbs.PBSRequest, error) {
	acctId, err := toAccountId(req)
	if err != nil {
		return nil, err
	}

	tId, err := toTransactionId(req)
	if err != nil {
		return nil, err
	}

	isSecure, err := toSecure(req)
	if err != nil {
		return nil, err
	}

	isDebug := false
	if req.Test == 1 {
		isDebug = true
	}

	url := ""
	domain := ""
	if req.Site != nil {
		url = req.Site.Page
		domain = req.Site.Domain
	}

	cookie := pbs.NewPBSCookie()
	if req.User != nil {
		if req.User.BuyerUID != "" {
			cookie.TrySync(bidder.adapter.FamilyName(), req.User.BuyerUID)
		}

		// This shouldn't be appnexus-specific... but this line does correctly invert the
		// logic from adapters/openrtb_util.go, which will preserve this questionable behavior in legacy adapters.
		if req.User.ID != "" {
			cookie.TrySync("adnxs", req.User.ID)
		}
	}

	return &pbs.PBSRequest{
		AccountID: acctId,
		Tid:       tId,
		// CacheMarkup is excluded because no legacy adapters read from it
		// SortBids is excluded because no legacy adapters read from it
		// MaxKeyLength is excluded because no legacy adapters read from it
		Secure:        isSecure,
		TimeoutMillis: req.TMax,
		// AdUnits is excluded because no legacy adapters read from it
		IsDebug: isDebug,
		App:     req.App,
		Device:  req.Device,
		// PBSUser is excluded because rubicon is the only adapter which reads from it, and they're supporting OpenRTB directly
		// SDK is excluded because that information doesn't exist in OpenRTB.
		// Bidders is excluded because no legacy adapters read from it
		User:   req.User,
		Cookie: cookie,
		Url:    url,
		Domain: domain,
		// Start is excluded because no legacy adapters read from it
	}, nil
}

func toAccountId(req *openrtb.BidRequest) (string, error) {
	if req.Site != nil && req.Site.Publisher != nil {
		return req.Site.Publisher.ID, nil
	}
	if req.App != nil && req.App.Publisher != nil {
		return req.App.Publisher.ID, nil
	}
	return "", errors.New("bidrequest.site.publisher.id or bidrequest.app.publisher.id required for legacy bidders.")
}

func toTransactionId(req *openrtb.BidRequest) (string, error) {
	if req.Source != nil {
		return req.Source.TID, nil
	}
	return "", errors.New("bidrequest.source.tid required for legacy bidders.")
}

func toSecure(req *openrtb.BidRequest) (secure int8, err error) {
	secure = -1
	for _, imp := range req.Imp {
		if imp.Secure != nil {
			thisVal := *imp.Secure
			if thisVal == 0 {
				if secure == 1 {
					err = errors.New("bidrequest.imp[i].secure must be consistent for legacy bidders. Mixing 0 and 1 are not allowed.")
					return
				}
				secure = 0
			} else if thisVal == 1 {
				if secure == 0 {
					err = errors.New("bidrequest.imp[i].secure must be consistent for legacy bidders. Mixing 0 and 1 are not allowed.")
					return
				}
				secure = 1
			}
		}
	}
	if secure == -1 {
		secure = 0
	}

	return
}

func toLegacyBidder(req *openrtb.BidRequest, name openrtb_ext.BidderName) (*pbs.PBSBidder, []error) {
	adUnits, errs := toPBSAdUnits(req)
	if len(adUnits) > 0 {
		return &pbs.PBSBidder{
			BidderCode: string(name),
			// AdUnitCode is excluded because no legacy adapters read from it
			// ResponseTime is excluded because no legacy adapters read from it
			// NumBids is excluded because no legacy adapters read from it
			// Error is excluded because no legacy adapters read from it
			// NoCookie is excluded because no legacy adapters read from it
			// NoBid is excluded because no legacy adapters read from it
			// UsersyncInfo is excluded because no legacy adapters read from it
			// Debug is excluded because legacy adapters only use it in nil-safe ways.
			//   They *do* write to it, though, so it may be read when unpacking the response.
			AdUnits: adUnits,
		}, errs
	} else {
		return nil, errs
	}
}

func toPBSAdUnits(req *openrtb.BidRequest) ([]pbs.PBSAdUnit, []error) {
	adUnits := make([]pbs.PBSAdUnit, len(req.Imp))
	var errs []error = nil
	nextAdUnit := 0
	for i := 0; i < len(req.Imp); i++ {
		err := initPBSAdUnit(&(req.Imp[i]), &(adUnits[nextAdUnit]))
		if err != nil {
			errs = append(errs, err)
		} else {
			nextAdUnit++
		}
	}
	return adUnits[:nextAdUnit], errs
}

func initPBSAdUnit(imp *openrtb.Imp, adUnit *pbs.PBSAdUnit) error {
	video := pbs.PBSVideo{}
	if imp.Video != nil {
		video.Mimes = imp.Video.MIMEs
		video.Minduration = imp.Video.MinDuration
		video.Maxduration = imp.Video.MaxDuration
		if imp.Video.StartDelay != nil {
			video.Startdelay = int64(*imp.Video.StartDelay)
		}
		if imp.Video.Skip != nil {
			video.Skippable = int(*imp.Video.Skip)
		}
		if len(imp.Video.PlaybackMethod) == 1 {
			video.PlaybackMethod = int8(imp.Video.PlaybackMethod[0])
		}
		if len(imp.Video.Protocols) > 0 {
			video.Protocols = make([]int8, len(imp.Video.Protocols))
			for i := 0; i < len(imp.Video.Protocols); i++ {
				video.Protocols[i] = int8(imp.Video.Protocols[i])
			}
		}
	}
	topFrame := int8(0)
	var sizes []openrtb.Format = nil
	if imp.Banner != nil {
		topFrame = imp.Banner.TopFrame
		sizes = imp.Banner.Format
	}

	params, _, _, err := jsonparser.Get(imp.Ext, "bidder")
	if err != nil {
		return err
	}

	mediaTypes := make([]pbs.MediaType, 0, 2)
	if imp.Banner != nil {
		mediaTypes = append(mediaTypes, pbs.MEDIA_TYPE_BANNER)
	}
	if imp.Video != nil {
		mediaTypes = append(mediaTypes, pbs.MEDIA_TYPE_VIDEO)
	}
	if len(mediaTypes) == 0 {
		return errors.New("legacy bidders can only bid on banner and video ad units")
	}

	adUnit.Sizes = sizes
	adUnit.TopFrame = topFrame
	adUnit.Code = imp.ID
	adUnit.BidID = imp.ID
	adUnit.Params = json.RawMessage(params)
	adUnit.Video = video
	adUnit.MediaTypes = mediaTypes
	adUnit.Instl = imp.Instl

	return nil
}

// ----------------------------------------------------------------------------
// Response transformations.

// toNewResponse is a best-effort transformation of legacy Bids into an OpenRTB response.
func toNewResponse(bids pbs.PBSBidSlice, bidder *pbs.PBSBidder, bidderTarg *targetData, name openrtb_ext.BidderName) (*pbsOrtbSeatBid, []error) {
	newBids, errs := transformBids(bids, bidderTarg, name)
	return &pbsOrtbSeatBid{
		bids:      newBids,
		httpCalls: transformDebugs(bidder.Debug),
	}, errs
}

func transformBids(legacyBids pbs.PBSBidSlice, bidderTarg *targetData, name openrtb_ext.BidderName) ([]*pbsOrtbBid, []error) {
	newBids := make([]*pbsOrtbBid, 0, len(legacyBids))
	var errs []error = nil
	for _, legacyBid := range legacyBids {
		if legacyBid != nil {
			newBid, err := transformBid(legacyBid, bidderTarg, name)
			if err == nil {
				newBids = append(newBids, newBid)
			} else {
				errs = append(errs, err)
			}
		}
	}
	return newBids, errs
}

func transformBid(legacyBid *pbs.PBSBid, bidderTarg *targetData, name openrtb_ext.BidderName) (*pbsOrtbBid, error) {
	newBid := transformBidToOrtb(legacyBid)

	newBidType, err := openrtb_ext.ParseBidType(legacyBid.CreativeMediaType)
	if err != nil {
		return nil, err
	}

	targets, err := bidderTarg.makePrebidTargets(name, newBid)
	if targets != nil {
		return nil, err
	}

	return &pbsOrtbBid{
		bid:        newBid,
		bidType:    newBidType,
		bidTargets: targets,
	}, nil
}

func transformBidToOrtb(legacyBid *pbs.PBSBid) *openrtb.Bid {
	return &openrtb.Bid{
		ID:    legacyBid.BidID,
		ImpID: legacyBid.AdUnitCode,
		CrID:  legacyBid.Creative_id,
		// legacyBid.CreativeMediaType is handled by transformBid(), because it doesn't exist on the openrtb.Bid
		// legacyBid.BidderCode is handled by the exchange, which already knows which bidder we are.
		// legacyBid.BidHash is ignored, because it doesn't get sent in the response anyway
		Price:  legacyBid.Price,
		NURL:   legacyBid.NURL,
		AdM:    legacyBid.Adm,
		W:      legacyBid.Width,
		H:      legacyBid.Height,
		DealID: legacyBid.DealId,
		// TODO #216: Support CacheID here
		// TODO: #216: Support CacheURL here
		// ResponseTime is handled by the exchange, since it doesn't exist in the OpenRTB Bid
		// AdServerTargeting is handled by the exchange. Rubicon's adapter is the only one which writes to it,
		//   but that doesn't matter since they're supporting OpenRTB directly.
	}
}

func transformDebugs(legacyDebugs []*pbs.BidderDebug) []*openrtb_ext.ExtHttpCall {
	newDebug := make([]*openrtb_ext.ExtHttpCall, 0, len(legacyDebugs))
	for _, legacyDebug := range legacyDebugs {
		if legacyDebug != nil {
			newDebug = append(newDebug, transformDebug(legacyDebug))
		}
	}
	return newDebug
}

func transformDebug(legacyDebug *pbs.BidderDebug) *openrtb_ext.ExtHttpCall {
	return &openrtb_ext.ExtHttpCall{
		Uri:          legacyDebug.RequestURI,
		RequestBody:  legacyDebug.RequestBody,
		ResponseBody: legacyDebug.ResponseBody,
		Status:       legacyDebug.StatusCode,
	}
}
