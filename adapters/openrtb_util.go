package adapters

import (
	"github.com/prebid/prebid-server/pbs"

	"github.com/prebid/openrtb"
)

func makeOpenRTBGeneric(req *pbs.PBSRequest, bidder *pbs.PBSBidder, bidderFamily string, mediatypes []MediaType, singleMediaTypeImp bool) openrtb.BidRequest {

	imps := make([]openrtb.Imp, len(bidder.AdUnits)*len(mediatypes))
	ind := 0
	for i, unit := range bidder.AdUnits {
		if len(unit.Sizes) <= 0 {
			continue
		}

		if singleMediaTypeImp {
			for _, mType := range mediatypes {
				newImp := openrtb.Imp{
					ID:     unit.Code,
					Secure: req.Secure,
				}
				switch mType {
				case BANNER:
					newImp.Banner = &openrtb.Banner{
						W:        unit.Sizes[0].W,
						H:        unit.Sizes[0].H,
						Format:   unit.Sizes,
						TopFrame: unit.TopFrame,
					}
				case VIDEO:
					mimes := make([]string, len(req.Video.mimes))
					copy(mimes, req.Video.mimes)
					newImp.Video = &openrtb.Video{
						MIMEs:          mimes,
						MinDuration:    req.Video.Minduration,
						W:              unit.Sizes[0].W,
						H:              unit.Sizes[0].H,
						StartDelay:     req.Video.Startdelay,
						PlaybackMethod: req.Video.PlaybackMethod,
					}
				default:
					// Error - unknown media type
				}
				imps[ind] = newImp
				ind = ind + 1
			}
		} else {
			newImp := openrtb.Imp{
				ID:     unit.Code,
				Secure: req.Secure,
			}
			for _, mType := range mediatypes {
				switch mType {
				case BANNER:
					newImp.Banner = &openrtb.Banner{
						W:        unit.Sizes[0].W,
						H:        unit.Sizes[0].H,
						Format:   unit.Sizes,
						TopFrame: unit.TopFrame,
					}
				case VIDEO:
					mimes := make([]string, len(req.Video.mimes))
					copy(mimes, req.Video.mimes)
					newImp.Video = &openrtb.Video{
						MIMEs:          mimes,
						MinDuration:    req.Video.Minduration,
						W:              unit.Sizes[0].W,
						H:              unit.Sizes[0].H,
						StartDelay:     req.Video.Startdelay,
						PlaybackMethod: req.Video.PlaybackMethod,
					}
				default:
					// Error - unknown media type
				}
			}
			imps[i] = newImp
		}
	}

	if req.App != nil {
		return openrtb.BidRequest{
			ID:     req.Tid,
			Imp:    imps,
			App:    req.App,
			Device: req.Device,
			User:   req.User,
			Source: &openrtb.Source{
				TID: req.Tid,
			},
			AT:   1,
			TMax: req.TimeoutMillis,
		}
	}

	return openrtb.BidRequest{
		ID:  req.Tid,
		Imp: imps,
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
