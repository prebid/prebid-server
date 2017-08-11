package adapters

import (
	"github.com/prebid/prebid-server/pbs"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/openrtb"
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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test")

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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test")
	assert.Equal(t, resp.Imp[0].ID, "")
}

func TestOpenRTBMobile(t *testing.T) {
	pbReq := pbs.PBSRequest{
		AccountID:     "test_account_id",
		Tid:           "test_tid",
		CacheMarkup:   1,
		SortBids:      1,
		MaxKeyLength:  20,
		Secure:        1,
		TimeoutMillis: 1000,
		App: &openrtb.App{
			Bundle: "AppNexus.PrebidMobileDemo",
			Publisher: &openrtb.Publisher{
				ID: "1995257847363113",
			},
		},
		Device: &openrtb.Device{
			UA:    "test_ua",
			IP:    "test_ip",
			Make:  "test_make",
			Model: "test_model",
			IFA:   "test_ifa",
		},
		User: &openrtb.User{
			BuyerUID: "test_buyeruid",
		},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code: "unitCode",
				Sizes: []openrtb.Format{
					{
						W: 300,
						H: 250,
					},
				},
			},
		},
	}
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test")

	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Banner.W, 300)
	assert.EqualValues(t, resp.Imp[0].Banner.H, 250)

	assert.EqualValues(t, resp.App.Bundle, "AppNexus.PrebidMobileDemo")
	assert.EqualValues(t, resp.App.Publisher.ID, "1995257847363113")
	assert.EqualValues(t, resp.User.BuyerUID, "test_buyeruid")

	assert.EqualValues(t, resp.Device.UA, "test_ua")
	assert.EqualValues(t, resp.Device.IP, "test_ip")
	assert.EqualValues(t, resp.Device.Make, "test_make")
	assert.EqualValues(t, resp.Device.Model, "test_model")
	assert.EqualValues(t, resp.Device.IFA, "test_ifa")
}
