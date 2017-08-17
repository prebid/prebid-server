package adapters

import (
	"github.com/prebid/prebid-server/pbs"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/constants"
)

func TestOpenRTB(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code: "unitCode",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
			},
		},
	}
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, constants.FNRubicon)

	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Banner.W, 10)
	assert.EqualValues(t, resp.Imp[0].Banner.H, 12)
}

func TestOpenRTBNoSize(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code: "unitCode",
			},
		},
	}
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, constants.FNIndex)
	assert.Equal(t, resp.Imp[0].ID, "")
}
