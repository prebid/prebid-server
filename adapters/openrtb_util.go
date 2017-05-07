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
			// pmp
			// ext
		}
		if req.Protocol == "https" {
			imps[i].Secure = 1
		}
	}

	cur := make([]string, 1)
	cur[0] = "USD"

	return openrtb.BidRequest{
		ID:  req.Tid,
		Imp: imps,
		Cur: cur,
		Site: &openrtb.Site{
			Domain: req.Domain,
			Page:   req.Url,
		},
		Device: &openrtb.Device{
			UA: req.UserAgent,
			IP: req.IPAddress,
			// language? screen size? device type?
		},
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
