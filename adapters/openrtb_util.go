package adapters

import (
	"github.com/prebid/prebid-server/pbs"

	"github.com/prebid/openrtb"
)

func mediaTypeInSlice(t pbs.MediaType, list []pbs.MediaType) bool {
	for _, b := range list {
		if b == t {
			return true
		}
	}
	return false
}

func makeOpenRTBGeneric(req *pbs.PBSRequest, bidder *pbs.PBSBidder, bidderFamily string, mediatypes []pbs.MediaType, singleMediaTypeImp bool) openrtb.BidRequest {

	imps := make([]openrtb.Imp, len(bidder.AdUnits)*len(mediatypes))
	ind := 0
	impsPresent := false
	for _, unit := range bidder.AdUnits {
		if len(unit.Sizes) <= 0 {
			ind = ind + 1
			continue
		}

		if singleMediaTypeImp {
			for _, mType := range unit.MediaTypes {
				var newImp openrtb.Imp
				if mediaTypeInSlice(mType, mediatypes) {
					newImp = openrtb.Imp{
						ID:     unit.Code,
						Secure: req.Secure,
					}
					switch mType {
					case pbs.MEDIA_TYPE_BANNER:
						newImp.Banner = &openrtb.Banner{
							W:        unit.Sizes[0].W,
							H:        unit.Sizes[0].H,
							Format:   unit.Sizes,
							TopFrame: unit.TopFrame,
						}
					case pbs.MEDIA_TYPE_VIDEO:
						mimes := make([]string, len(unit.Video.Mimes))
						copy(mimes, unit.Video.Mimes)
						pbm := make([]int8, 1)
						pbm[0] = unit.Video.PlaybackMethod
						newImp.Video = &openrtb.Video{
							MIMEs:          mimes,
							MinDuration:    unit.Video.Minduration,
							W:              unit.Sizes[0].W,
							H:              unit.Sizes[0].H,
							StartDelay:     unit.Video.Startdelay,
							PlaybackMethod: pbm,
						}
					default:
						// Error - unknown media type
					}
					imps[ind] = newImp
					ind = ind + 1
					impsPresent = true
				}
			}
		} else {
			newImp := openrtb.Imp{
				ID:     unit.Code,
				Secure: req.Secure,
			}
			for _, mType := range unit.MediaTypes {
				switch mType {
				case pbs.MEDIA_TYPE_BANNER:
					newImp.Banner = &openrtb.Banner{
						W:        unit.Sizes[0].W,
						H:        unit.Sizes[0].H,
						Format:   unit.Sizes,
						TopFrame: unit.TopFrame,
					}
				case pbs.MEDIA_TYPE_VIDEO:
					mimes := make([]string, len(unit.Video.Mimes))
					copy(mimes, unit.Video.Mimes)
					pbm := make([]int8, 1)
					pbm[0] = unit.Video.PlaybackMethod
					newImp.Video = &openrtb.Video{
						MIMEs:          mimes,
						MinDuration:    unit.Video.Minduration,
						W:              unit.Sizes[0].W,
						H:              unit.Sizes[0].H,
						StartDelay:     unit.Video.Startdelay,
						PlaybackMethod: pbm,
					}
				default:
					// Error - unknown media type
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
			Source: &openrtb.Source{
				TID: req.Tid,
			},
			AT:   1,
			TMax: req.TimeoutMillis,
		}
	}

	return openrtb.BidRequest{
		ID:  req.Tid,
		Imp: newImps,
		Site: &openrtb.Site{
			Domain: req.Domain,
			Page:   req.Url,
		},
		Device: req.Device,
		User: &openrtb.User{
			BuyerUID: req.GetUserID(bidderFamily),
			ID:       req.GetUserID("adnxs"),
		},
		Source: &openrtb.Source{
			FD:  1, // upstream, aka header
			TID: req.Tid,
		},
		AT:   1,
		TMax: req.TimeoutMillis,
	}
}
