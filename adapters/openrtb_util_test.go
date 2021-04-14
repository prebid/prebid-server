package adapters

import (
	"testing"

	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"
	"github.com/stretchr/testify/assert"
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
				Sizes: []openrtb2.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Instl: 1,
			},
		},
	}
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})

	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, *resp.Imp[0].Banner.W, 10)
	assert.EqualValues(t, *resp.Imp[0].Banner.H, 12)
	assert.EqualValues(t, resp.Imp[0].Instl, 1)

	assert.Nil(t, resp.User.Ext)
	assert.Nil(t, resp.Regs)
}

func TestOpenRTBVideo(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO},
				Sizes: []openrtb2.Format{
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
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO})

	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, resp.Imp[0].Video.MaxDuration, 30)
	assert.EqualValues(t, resp.Imp[0].Video.MinDuration, 15)
	assert.EqualValues(t, *resp.Imp[0].Video.StartDelay, openrtb2.StartDelay(5))
	assert.EqualValues(t, resp.Imp[0].Video.PlaybackMethod, []openrtb2.PlaybackMethod{openrtb2.PlaybackMethod(1)})
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
				Sizes: []openrtb2.Format{
					{
						W: 10,
						H: 12,
					},
				},
			},
		},
	}
	_, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO})

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
				Sizes: []openrtb2.Format{
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
			{
				Code:       "unitCode2",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb2.Format{
					{
						W: 10,
						H: 12,
					},
				},
			},
		},
	}
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})
	assert.Equal(t, err, nil)
	for i := 0; i < len(resp.Imp); i++ {
		if resp.Imp[i].Video != nil {
			t.Errorf("No video impressions should exist.")
		}
	}
}

func TestOpenRTBMultiMediaImp(t *testing.T) {

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb2.Format{
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
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(resp.Imp), 1)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, *resp.Imp[0].Banner.W, 10)
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
				Sizes: []openrtb2.Format{
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
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(resp.Imp), 1)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, *resp.Imp[0].Banner.W, 10)
	assert.EqualValues(t, resp.Imp[0].Video, (*openrtb2.Video)(nil))
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
	_, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})
	if err == nil {
		t.Errorf("Bids without impressions should not be allowed.")
	}
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
		App: &openrtb2.App{
			Bundle: "AppNexus.PrebidMobileDemo",
			Publisher: &openrtb2.Publisher{
				ID: "1995257847363113",
			},
		},
		Device: &openrtb2.Device{
			UA:    "test_ua",
			IP:    "test_ip",
			Make:  "test_make",
			Model: "test_model",
			IFA:   "test_ifa",
		},
		User: &openrtb2.User{
			BuyerUID: "test_buyeruid",
		},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb2.Format{
					{
						W: 300,
						H: 250,
					},
				},
			},
		},
	}
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, *resp.Imp[0].Banner.W, 300)
	assert.EqualValues(t, *resp.Imp[0].Banner.H, 250)

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
		User: &openrtb2.User{},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode2",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb2.Format{
					{
						W: 10,
						H: 12,
					},
				},
			},
		},
	}
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})
	assert.Equal(t, err, nil)
	assert.EqualValues(t, resp.User, &openrtb2.User{})
}

func TestOpenRTBUserWithCookie(t *testing.T) {
	pbsCookie := usersync.NewPBSCookie()
	pbsCookie.TrySync("test", "abcde")
	pbReq := pbs.PBSRequest{
		User: &openrtb2.User{},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb2.Format{
					{
						W: 300,
						H: 250,
					},
				},
			},
		},
	}
	pbReq.Cookie = pbsCookie
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})
	assert.Equal(t, err, nil)
	assert.EqualValues(t, resp.User.BuyerUID, "abcde")
}

func TestSizesCopy(t *testing.T) {
	formats := []openrtb2.Format{
		{
			W: 10,
		},
		{
			Ext: []byte{0x5},
		},
	}
	clone := copyFormats(formats)

	if len(clone) != 2 {
		t.Error("The copy should have 2 elements")
	}
	if clone[0].W != 10 {
		t.Error("The Format's width should be preserved.")
	}
	if len(clone[1].Ext) != 1 || clone[1].Ext[0] != 0x5 {
		t.Error("The Format's Ext should be preserved.")
	}
	if &formats[0] == &clone[0] || &formats[1] == &clone[1] {
		t.Error("The Format elements should not point to the same instance")
	}
	if &formats[0] == &clone[0] || &formats[1] == &clone[1] {
		t.Error("The Format elements should not point to the same instance")
	}
	if &formats[1].Ext[0] == &clone[1].Ext[0] {
		t.Error("The Format.Ext property should point to two different instances")
	}
}

func TestMakeVideo(t *testing.T) {
	adUnit := pbs.PBSAdUnit{
		Code:       "unitCode",
		MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO},
		Sizes: []openrtb2.Format{
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
			Protocols:      []int8{1, 2, 5, 6},
		},
	}
	video := makeVideo(adUnit)
	assert.EqualValues(t, video.MinDuration, 15)
	assert.EqualValues(t, video.MaxDuration, 30)
	assert.EqualValues(t, *video.StartDelay, openrtb2.StartDelay(5))
	assert.EqualValues(t, len(video.PlaybackMethod), 1)
	assert.EqualValues(t, len(video.Protocols), 4)
}

func TestGDPR(t *testing.T) {

	rawUserExt := json.RawMessage(`{"consent": "12345"}`)
	userExt, _ := json.Marshal(rawUserExt)

	rawRegsExt := json.RawMessage(`{"gdpr": 1}`)
	regsExt, _ := json.Marshal(rawRegsExt)

	pbReq := pbs.PBSRequest{
		User: &openrtb2.User{
			Ext: userExt,
		},
		Regs: &openrtb2.Regs{
			Ext: regsExt,
		},
	}

	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb2.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Instl: 1,
			},
		},
	}
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})

	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, *resp.Imp[0].Banner.W, 10)
	assert.EqualValues(t, *resp.Imp[0].Banner.H, 12)
	assert.EqualValues(t, resp.Imp[0].Instl, 1)

	assert.EqualValues(t, resp.User.Ext, userExt)
	assert.EqualValues(t, resp.Regs.Ext, regsExt)
}

func TestGDPRMobile(t *testing.T) {
	rawUserExt := json.RawMessage(`{"consent": "12345"}`)
	userExt, _ := json.Marshal(rawUserExt)

	rawRegsExt := json.RawMessage(`{"gdpr": 1}`)
	regsExt, _ := json.Marshal(rawRegsExt)

	pbReq := pbs.PBSRequest{
		AccountID:     "test_account_id",
		Tid:           "test_tid",
		CacheMarkup:   1,
		SortBids:      1,
		MaxKeyLength:  20,
		Secure:        1,
		TimeoutMillis: 1000,
		App: &openrtb2.App{
			Bundle: "AppNexus.PrebidMobileDemo",
			Publisher: &openrtb2.Publisher{
				ID: "1995257847363113",
			},
		},
		Device: &openrtb2.Device{
			UA:    "test_ua",
			IP:    "test_ip",
			Make:  "test_make",
			Model: "test_model",
			IFA:   "test_ifa",
		},
		User: &openrtb2.User{
			BuyerUID: "test_buyeruid",
			Ext:      userExt,
		},
		Regs: &openrtb2.Regs{
			Ext: regsExt,
		},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb2.Format{
					{
						W: 300,
						H: 250,
					},
				},
			},
		},
	}
	resp, err := MakeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER})
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.EqualValues(t, *resp.Imp[0].Banner.W, 300)
	assert.EqualValues(t, *resp.Imp[0].Banner.H, 250)

	assert.EqualValues(t, resp.App.Bundle, "AppNexus.PrebidMobileDemo")
	assert.EqualValues(t, resp.App.Publisher.ID, "1995257847363113")
	assert.EqualValues(t, resp.User.BuyerUID, "test_buyeruid")

	assert.EqualValues(t, resp.Device.UA, "test_ua")
	assert.EqualValues(t, resp.Device.IP, "test_ip")
	assert.EqualValues(t, resp.Device.Make, "test_make")
	assert.EqualValues(t, resp.Device.Model, "test_model")
	assert.EqualValues(t, resp.Device.IFA, "test_ifa")

	assert.EqualValues(t, resp.User.Ext, userExt)
	assert.EqualValues(t, resp.Regs.Ext, regsExt)
}
