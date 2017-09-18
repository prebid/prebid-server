package adapters

import (
	"github.com/prebid/prebid-server/pbs"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mxmCherry/openrtb"
)

func TestCommonMediaTypes(t *testing.T) {
	mt1 := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	mt2 := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	common := commonMediaTypes(mt1, mt2)
	assert.Equal(t, len(common), 1)
	assert.Equal(t, common[0], pbs.MEDIA_TYPE_BANNER)

	common2 := commonMediaTypes(mt2, mt1)
	assert.Equal(t, len(common2), 1)
	assert.Equal(t, common2[0], pbs.MEDIA_TYPE_BANNER)

	mt3 := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	mt4 := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	common3 := commonMediaTypes(mt3, mt4)
	assert.Equal(t, len(common3), 2)

	mt5 := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	mt6 := []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}
	common4 := commonMediaTypes(mt5, mt6)
	assert.Equal(t, len(common4), 0)
}

func TestOpenRTB(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)

	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Banner.W, 10)
	assert.EqualValues(t, resp.Imp[0].Banner.H, 12)
}

func TestOpenRTBVideo(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO},
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Video: pbs.PBSVideo{
					Mimes:          []string{"video/mp4"},
					Minduration:    15,
					Maxduration:    30,
					Startdelay:     5,
					Skippable:      0,
					PlaybackMethod: 1,
				},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}, true)

	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Video.MaxDuration, 30)
	assert.EqualValues(t, resp.Imp[0].Video.MinDuration, 15)
	assert.EqualValues(t, resp.Imp[0].Video.StartDelay, 5)
	assert.EqualValues(t, resp.Imp[0].Video.PlaybackMethod, []int8{1})
	assert.EqualValues(t, resp.Imp[0].Video.MIMEs, []string{"video/mp4"})
}

func TestOpenRTBVideoNoVideoData(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO},
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
			},
		},
	}
	_, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}, true)

	assert.NotEqual(t, err, nil)

}

func TestOpenRTBVideoFilteredOut(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO},
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Video: pbs.PBSVideo{
					Mimes:          []string{"video/mp4"},
					Minduration:    15,
					Maxduration:    30,
					Startdelay:     5,
					Skippable:      0,
					PlaybackMethod: 1,
				},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp, []openrtb.Imp(nil))
}

func TestOpenRTBMultiMediaImp(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Video: pbs.PBSVideo{
					Mimes:          []string{"video/mp4"},
					Minduration:    15,
					Maxduration:    30,
					Startdelay:     5,
					Skippable:      0,
					PlaybackMethod: 1,
				},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER}, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(resp.Imp), 1)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Banner.W, 10)
	assert.EqualValues(t, resp.Imp[0].Video.W, 10)
	assert.EqualValues(t, resp.Imp[0].Video.MaxDuration, 30)
	assert.EqualValues(t, resp.Imp[0].Video.MinDuration, 15)
}

func TestOpenRTBMultiMediaImpFiltered(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Video: pbs.PBSVideo{
					Mimes:          []string{"video/mp4"},
					Minduration:    15,
					Maxduration:    30,
					Startdelay:     5,
					Skippable:      0,
					PlaybackMethod: 1,
				},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(resp.Imp), 1)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Banner.W, 10)
	assert.EqualValues(t, resp.Imp[0].Video, (*openrtb.Video)(nil))
}

func TestOpenRTBSingleMediaImp(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Video: pbs.PBSVideo{
					Mimes:          []string{"video/mp4"},
					Minduration:    15,
					Maxduration:    30,
					Startdelay:     5,
					Skippable:      0,
					PlaybackMethod: 1,
				},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER}, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(resp.Imp), 2)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Video.MaxDuration, 30)
	assert.EqualValues(t, resp.Imp[0].Video.MinDuration, 15)
	assert.Equal(t, resp.Imp[1].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[1].Banner.W, 10)
}

func TestOpenRTBNoSize(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp, []openrtb.Imp(nil))
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
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 300,
						H: 250,
					},
				},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)
	assert.Equal(t, err, nil)
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

func TestOpenRTBEmptyUser(t *testing.T) {
	pbReq := pbs.PBSRequest{
		User: &openrtb.User{},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
			},
		},
	}
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)
	assert.Equal(t, err, nil)
	assert.EqualValues(t, resp.User, &openrtb.User{})
}

func TestOpenRTBUserWithCookie(t *testing.T) {
	pbsCookie := pbs.NewPBSCookie()
	pbsCookie.TrySync("test", "abcde")
	pbReq := pbs.PBSRequest{
		User: &openrtb.User{},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
			},
		},
	}
	pbReq.Cookie = pbsCookie
	resp, err := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)
	assert.Equal(t, err, nil)
	assert.EqualValues(t, resp.User.BuyerUID, "abcde")
}
