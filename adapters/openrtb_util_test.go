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
				Video: pbs.PBSVideo {
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
				Video: pbs.PBSVideo {
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
				Video: pbs.PBSVideo {
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
				Video: pbs.PBSVideo {
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
