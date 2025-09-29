package ortb2blocking

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateBAttr_DoesNotCreateMediaTypeObjects(t *testing.T) {
	cfg := config{
		Attributes: Attributes{
			Battr: Battr{
				BlockedBannerAttr: []int{1, 2, 3},
				BlockedVideoAttr:  []int{4, 5, 6},
				BlockedAudioAttr:  []int{7, 8, 9},
			},
		},
	}

	var sizeW int64 = 640
	var sizeH int64 = 480

	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp-video-only",
						Video: &openrtb2.Video{W: &sizeW, H: &sizeH},
						// Banner = nil, Audio = nil
					},
					{
						ID:    "imp-audio-only",
						Audio: &openrtb2.Audio{},
						// Banner = nil, Video = nil
					},
				},
			},
		},
		Bidder: "test-bidder",
	}

	var blockingAttrs blockingAttributes
	var result hookstage.HookResult[hookstage.BidderRequestPayload]
	var changeSet hookstage.ChangeSet[hookstage.BidderRequestPayload]

	err := updateBAttr(cfg, payload, &blockingAttrs, &result, &changeSet)
	require.NoError(t, err)

	// Check that mutations were created only for existing media types
	mutations := changeSet.Mutations()

	// Check the number of mutations - there should be only one for video and one for audio
	videoMutationsCount := 0
	audioMutationsCount := 0
	bannerMutationsCount := 0

	for _, mutation := range mutations {
		if len(mutation.Key()) >= 3 {
			mediaType := mutation.Key()[2] // the third path element must be a media type
			switch mediaType {
			case "video":
				videoMutationsCount++
			case "audio":
				audioMutationsCount++
			case "banner":
				bannerMutationsCount++
			}
		}
	}

	assert.Equal(t, 1, videoMutationsCount, "Should have exactly one video mutation")
	assert.Equal(t, 1, audioMutationsCount, "Should have exactly one audio mutation")
	assert.Equal(t, 0, bannerMutationsCount, "Should have no banner mutations")

	// Apply mutations manually using functions from the code
	for _, mutation := range mutations {
		updatedPayload, err := mutation.Apply(payload)
		require.NoError(t, err)
		payload = updatedPayload
	}

	// Check that Banner/Audio objects were NOT created
	for _, imp := range payload.Request.BidRequest.Imp {
		switch imp.ID {
		case "imp-video-only":
			assert.NotNil(t, imp.Video, "Video should exist")
			assert.Nil(t, imp.Banner, "Banner should NOT be created")
			assert.Nil(t, imp.Audio, "Audio should NOT be created")
			// Check that battr was added to video
			if imp.Video != nil {
				assert.Len(t, imp.Video.BAttr, 3, "Video should have battr applied")
			}
		case "imp-audio-only":
			assert.NotNil(t, imp.Audio, "Audio should exist")
			assert.Nil(t, imp.Banner, "Banner should NOT be created")
			assert.Nil(t, imp.Video, "Video should NOT be created")
			// Check that battr was added to audio
			if imp.Audio != nil {
				assert.Len(t, imp.Audio.BAttr, 3, "Audio should have battr applied")
			}
		}
	}
}

func TestUpdateBAttr_AppliesOnlyToExistingMediaTypes(t *testing.T) {
	cfg := config{
		Attributes: Attributes{
			Battr: Battr{
				BlockedBannerAttr: []int{1, 2},
				BlockedVideoAttr:  []int{3, 4},
				BlockedAudioAttr:  []int{5, 6},
			},
		},
	}

	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp-banner-only",
						Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)},
					},
					{
						ID:     "imp-mixed",
						Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](728), H: ptrutil.ToPtr[int64](90)},
						Video:  &openrtb2.Video{W: ptrutil.ToPtr[int64](640), H: ptrutil.ToPtr[int64](480)},
					},
				},
			},
		},
		Bidder: "test-bidder",
	}

	var blockingAttrs blockingAttributes
	var result hookstage.HookResult[hookstage.BidderRequestPayload]
	var changeSet hookstage.ChangeSet[hookstage.BidderRequestPayload]

	err := updateBAttr(cfg, payload, &blockingAttrs, &result, &changeSet)
	require.NoError(t, err)

	// We apply mutations
	mutations := changeSet.Mutations()
	for _, mutation := range mutations {
		updatedPayload, err := mutation.Apply(payload)
		require.NoError(t, err)
		payload = updatedPayload
	}

	for _, imp := range payload.Request.BidRequest.Imp {
		switch imp.ID {
		case "imp-banner-only":
			assert.NotNil(t, imp.Banner, "Banner should exist")
			assert.Nil(t, imp.Video, "Video should NOT be created")
			assert.Nil(t, imp.Audio, "Audio should NOT be created")
			if imp.Banner != nil {
				assert.Len(t, imp.Banner.BAttr, 2, "Banner should have battr applied")
			}
		case "imp-mixed":
			assert.NotNil(t, imp.Banner, "Banner should exist")
			assert.NotNil(t, imp.Video, "Video should exist")
			assert.Nil(t, imp.Audio, "Audio should NOT be created")
			if imp.Banner != nil {
				assert.Len(t, imp.Banner.BAttr, 2, "Banner should have battr applied")
			}
			if imp.Video != nil {
				assert.Len(t, imp.Video.BAttr, 2, "Video should have battr applied")
			}
		}
	}
}

func TestFilterByMediaType(t *testing.T) {
	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "has-banner",
						Banner: &openrtb2.Banner{},
					},
					{
						ID: "no-banner",
						// Banner = nil
					},
				},
			},
		},
	}

	overrides := map[string][]int{
		"has-banner": {1, 2, 3},
		"no-banner":  {4, 5, 6},
	}

	filtered := filterByMediaType(payload, overrides, func(imp openrtb2.Imp) bool {
		return imp.Banner != nil
	})

	// Only the impression with the Banner should remain.
	assert.Len(t, filtered, 1)
	assert.Contains(t, filtered, "has-banner")
	assert.NotContains(t, filtered, "no-banner")
	assert.Equal(t, []int{1, 2, 3}, filtered["has-banner"])
}

func TestCreateBAttrMutation(t *testing.T) {
	bAttrByImp := map[string][]int{
		"imp1": {1, 2},
		"imp2": {3, 4},
	}

	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
					{
						ID:     "imp2",
						Banner: &openrtb2.Banner{},
					},
				},
			},
		},
	}

	mutation := createBAttrMutation(bAttrByImp, "banner")
	updatedPayload, err := mutation(payload)
	require.NoError(t, err)

	// Check that battr was applied
	for _, imp := range updatedPayload.Request.BidRequest.Imp {
		switch imp.ID {
		case "imp1":
			require.NotNil(t, imp.Banner)
			assert.Len(t, imp.Banner.BAttr, 2)
			assert.Equal(t, adcom1.CreativeAttribute(1), imp.Banner.BAttr[0])
			assert.Equal(t, adcom1.CreativeAttribute(2), imp.Banner.BAttr[1])
		case "imp2":
			require.NotNil(t, imp.Banner)
			assert.Len(t, imp.Banner.BAttr, 2)
			assert.Equal(t, adcom1.CreativeAttribute(3), imp.Banner.BAttr[0])
			assert.Equal(t, adcom1.CreativeAttribute(4), imp.Banner.BAttr[1])
		}
	}
}
