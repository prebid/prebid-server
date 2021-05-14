package adapters

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/pbs"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func mediaTypeInSlice(t pbs.MediaType, list []pbs.MediaType) bool {
	for _, b := range list {
		if b == t {
			return true
		}
	}
	return false
}

func commonMediaTypes(l1 []pbs.MediaType, l2 []pbs.MediaType) []pbs.MediaType {
	res := make([]pbs.MediaType, min(len(l1), len(l2)))
	i := 0
	for _, b := range l1 {
		if mediaTypeInSlice(b, l2) {
			res[i] = b
			i = i + 1
		}
	}
	return res[:i]
}

func makeBanner(unit pbs.PBSAdUnit) *openrtb2.Banner {
	return &openrtb2.Banner{
		W:        openrtb2.Int64Ptr(unit.Sizes[0].W),
		H:        openrtb2.Int64Ptr(unit.Sizes[0].H),
		Format:   copyFormats(unit.Sizes), // defensive copy because adapters may mutate Imps, and this is shared data
		TopFrame: unit.TopFrame,
	}
}

func makeVideo(unit pbs.PBSAdUnit) *openrtb2.Video {
	// empty mimes array is a sign of uninitialized Video object
	if len(unit.Video.Mimes) < 1 {
		return nil
	}
	mimes := make([]string, len(unit.Video.Mimes))
	copy(mimes, unit.Video.Mimes)
	pbm := make([]openrtb2.PlaybackMethod, 1)
	//this will become int8 soon, so we only care about the first index in the array
	pbm[0] = openrtb2.PlaybackMethod(unit.Video.PlaybackMethod)

	protocols := make([]openrtb2.Protocol, 0, len(unit.Video.Protocols))
	for _, protocol := range unit.Video.Protocols {
		protocols = append(protocols, openrtb2.Protocol(protocol))
	}
	return &openrtb2.Video{
		MIMEs:          mimes,
		MinDuration:    unit.Video.Minduration,
		MaxDuration:    unit.Video.Maxduration,
		W:              unit.Sizes[0].W,
		H:              unit.Sizes[0].H,
		StartDelay:     openrtb2.StartDelay(unit.Video.Startdelay).Ptr(),
		PlaybackMethod: pbm,
		Protocols:      protocols,
	}
}

// adapters.MakeOpenRTBGeneric makes an openRTB request from the PBS-specific structs.
//
// Any objects pointed to by the returned BidRequest *must not be mutated*, or we will get race conditions.
// The only exception is the Imp property, whose objects will be created new by this method and can be mutated freely.
func MakeOpenRTBGeneric(req *pbs.PBSRequest, bidder *pbs.PBSBidder, bidderFamily string, allowedMediatypes []pbs.MediaType) (openrtb2.BidRequest, error) {
	imps := make([]openrtb2.Imp, 0, len(bidder.AdUnits)*len(allowedMediatypes))
	for _, unit := range bidder.AdUnits {
		if len(unit.Sizes) <= 0 {
			continue
		}
		unitMediaTypes := commonMediaTypes(unit.MediaTypes, allowedMediatypes)
		if len(unitMediaTypes) == 0 {
			continue
		}

		newImp := openrtb2.Imp{
			ID:     unit.Code,
			Secure: &req.Secure,
			Instl:  unit.Instl,
		}
		for _, mType := range unitMediaTypes {
			switch mType {
			case pbs.MEDIA_TYPE_BANNER:
				newImp.Banner = makeBanner(unit)
			case pbs.MEDIA_TYPE_VIDEO:
				newImp.Video = makeVideo(unit)
				// It's strange to error here... but preserves legacy behavior in legacy code. See #603.
				if newImp.Video == nil {
					return openrtb2.BidRequest{}, &errortypes.BadInput{
						Message: "Invalid AdUnit: VIDEO media type with no video data",
					}
				}
			}
		}
		if newImp.Banner != nil || newImp.Video != nil {
			imps = append(imps, newImp)
		}
	}

	if len(imps) < 1 {
		return openrtb2.BidRequest{}, &errortypes.BadInput{
			Message: "openRTB bids need at least one Imp",
		}
	}

	if req.App != nil {
		return openrtb2.BidRequest{
			ID:     req.Tid,
			Imp:    imps,
			App:    req.App,
			Device: req.Device,
			User:   req.User,
			Source: &openrtb2.Source{
				TID: req.Tid,
			},
			AT:   1,
			TMax: req.TimeoutMillis,
			Regs: req.Regs,
		}, nil
	}

	buyerUID, _, _ := req.Cookie.GetUID(bidderFamily)
	id, _, _ := req.Cookie.GetUID("adnxs")

	var userExt json.RawMessage
	if req.User != nil {
		userExt = req.User.Ext
	}

	return openrtb2.BidRequest{
		ID:  req.Tid,
		Imp: imps,
		Site: &openrtb2.Site{
			Domain: req.Domain,
			Page:   req.Url,
		},
		Device: req.Device,
		User: &openrtb2.User{
			BuyerUID: buyerUID,
			ID:       id,
			Ext:      userExt,
		},
		Source: &openrtb2.Source{
			FD:  1, // upstream, aka header
			TID: req.Tid,
		},
		AT:   1,
		TMax: req.TimeoutMillis,
		Regs: req.Regs,
	}, nil
}

func copyFormats(sizes []openrtb2.Format) []openrtb2.Format {
	sizesCopy := make([]openrtb2.Format, len(sizes))
	for i := 0; i < len(sizes); i++ {
		sizesCopy[i] = sizes[i]
		sizesCopy[i].Ext = append([]byte(nil), sizes[i].Ext...)
	}
	return sizesCopy
}
