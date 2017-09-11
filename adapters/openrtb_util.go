package adapters

import (
	"github.com/prebid/prebid-server/pbs"

	"github.com/prebid/openrtb"
)

func makeOpenRTBGeneric(req *pbs.PBSRequest, bidder *pbs.PBSBidder, bidderFamily string) openrtb.BidRequest {

	imps := make([]openrtb.Imp, len(bidder.AdUnits))
	for i, unit := range bidder.AdUnits {
		if len(unit.Sizes) <= 0 {
			continue
		}

		imps[i] = openrtb.Imp{
			ID: unit.Code,
			Banner: &openrtb.Banner{
				W:        unit.Sizes[0].W,
				H:        unit.Sizes[0].H,
				Format:   unit.Sizes,
				TopFrame: unit.TopFrame,
			},
			Secure: req.Secure,
			// pmp
			// ext
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
