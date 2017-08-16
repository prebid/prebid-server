package adapters

import (
	"github.com/prebid/prebid-server/pbs"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/openrtb"
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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)

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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}, true)

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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}, true)

	assert.Equal(t, resp.Imp[0].ID, "unitCode")
	assert.Equal(t, resp.Imp[0].Video.MaxDuration, uint64(0x0))

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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)
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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER}, false)
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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, false)
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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO, pbs.MEDIA_TYPE_BANNER}, true)
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
	resp := makeOpenRTBGeneric(&pbReq, &pbBidder, "test", []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}, true)
	//	assert.Equal(t, resp.Imp[0].ID, "")
	assert.Equal(t, resp.Imp, []openrtb.Imp(nil))
}
