package adapters

import (
	"github.com/prebid/prebid-server/pbs"

	"errors"
	"github.com/mxmCherry/openrtb"
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

func makeBanner(unit pbs.PBSAdUnit) *openrtb.Banner {
	return &openrtb.Banner{
		W:        unit.Sizes[0].W,
		H:        unit.Sizes[0].H,
		Format:   unit.Sizes,
		TopFrame: unit.TopFrame,
	}
}

func makeVideo(unit pbs.PBSAdUnit) *openrtb.Video {
	// empty mimes array is a sign of uninitialized Video object
	if len(unit.Video.Mimes) < 1 {
		return nil
	}
	mimes := make([]string, len(unit.Video.Mimes))
	copy(mimes, unit.Video.Mimes)
	pbm := make([]int8, 1)
	pbm[0] = unit.Video.PlaybackMethod
	return &openrtb.Video{
		MIMEs:          mimes,
		MinDuration:    unit.Video.Minduration,
		MaxDuration:    unit.Video.Maxduration,
		W:              unit.Sizes[0].W,
		H:              unit.Sizes[0].H,
		StartDelay:     unit.Video.Startdelay,
		PlaybackMethod: pbm,
	}
}

func makeOpenRTBGeneric(req *pbs.PBSRequest, bidder *pbs.PBSBidder, bidderFamily string, allowedMediatypes []pbs.MediaType, singleMediaTypeImp bool) (openrtb.BidRequest, error) {

	imps := make([]openrtb.Imp, len(bidder.AdUnits)*len(allowedMediatypes))
	ind := 0
	impsPresent := false
	for _, unit := range bidder.AdUnits {
		if len(unit.Sizes) <= 0 {
			ind = ind + 1
			continue
		}
		unitMediaTypes := commonMediaTypes(unit.MediaTypes, allowedMediatypes)
		if len(unitMediaTypes) == 0 {
			continue
		}

		if singleMediaTypeImp {
			for _, mType := range unitMediaTypes {
				newImp := openrtb.Imp{
					ID:     unit.Code,
					Secure: &req.Secure,
				}
				switch mType {
				case pbs.MEDIA_TYPE_BANNER:
					newImp.Banner = makeBanner(unit)
				case pbs.MEDIA_TYPE_VIDEO:
					video := makeVideo(unit)
					if video == nil {
						return openrtb.BidRequest{}, errors.New("Invalid AdUnit: VIDEO media type with no video data")
					}
					newImp.Video = video
				default:
					// Error - unknown media type
					continue
				}
				imps[ind] = newImp
				ind = ind + 1
				impsPresent = true
			}
		} else {
			newImp := openrtb.Imp{
				ID:     unit.Code,
				Secure: &req.Secure,
			}
			for _, mType := range unitMediaTypes {
				switch mType {
				case pbs.MEDIA_TYPE_BANNER:
					newImp.Banner = makeBanner(unit)
				case pbs.MEDIA_TYPE_VIDEO:
					newImp.Video = makeVideo(unit)
				default:
					// Error - unknown media type
					continue
				}
			}
			imps[ind] = newImp
			ind = ind + 1
			impsPresent = true
		}
	}

	newImps := imps[:ind]
	if !impsPresent {
		newImps = nil
	}

	if req.App != nil {
		return openrtb.BidRequest{
			ID:     req.Tid,
			Imp:    newImps,
			App:    req.App,
			Device: req.Device,
			User:   req.User,
			Source: &openrtb.Source{
				TID: req.Tid,
			},
			AT:   1,
			TMax: req.TimeoutMillis,
		}, nil
	}

	buyerUID, _, _ := req.Cookie.GetUID(bidderFamily)
	id, _, _ := req.Cookie.GetUID("adnxs")

	return openrtb.BidRequest{
		ID:  req.Tid,
		Imp: newImps,
		Site: &openrtb.Site{
			Domain: req.Domain,
			Page:   req.Url,
		},
		Device: req.Device,
		User: &openrtb.User{
			BuyerUID: buyerUID,
			ID:       id,
		},
		Source: &openrtb.Source{
			FD:  1, // upstream, aka header
			TID: req.Tid,
		},
		AT:   1,
		TMax: req.TimeoutMillis,
	}, nil
}
