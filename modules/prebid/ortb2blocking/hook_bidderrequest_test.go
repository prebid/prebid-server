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

func TestCreateBTypeMutation(t *testing.T) {
	bTypeByImp := map[string][]int{
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

	mutation := createBTypeMutation(bTypeByImp)
	updatedPayload, err := mutation(payload)
	require.NoError(t, err)

	// Check that btype was applied
	for _, imp := range updatedPayload.Request.BidRequest.Imp {
		switch imp.ID {
		case "imp1":
			require.NotNil(t, imp.Banner)
			assert.Len(t, imp.Banner.BType, 2)
			assert.Equal(t, openrtb2.BannerAdType(1), imp.Banner.BType[0])
			assert.Equal(t, openrtb2.BannerAdType(2), imp.Banner.BType[1])
		case "imp2":
			require.NotNil(t, imp.Banner)
			assert.Len(t, imp.Banner.BType, 2)
			assert.Equal(t, openrtb2.BannerAdType(3), imp.Banner.BType[0])
			assert.Equal(t, openrtb2.BannerAdType(4), imp.Banner.BType[1])
		}
	}
}

func TestUpdateBType_DoesNotCreateBannerObjects(t *testing.T) {
	cfg := config{
		Attributes: Attributes{
			Btype: Btype{
				BlockedBannerType: []int{1, 2, 3},
			},
		},
	}

	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp-video-only",
						Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](640), H: ptrutil.ToPtr[int64](480)},
					},
					{
						ID:     "imp-with-banner",
						Banner: &openrtb2.Banner{},
					},
				},
			},
		},
		Bidder: "test-bidder",
	}

	var blockingAttrs blockingAttributes
	var result hookstage.HookResult[hookstage.BidderRequestPayload]
	var changeSet hookstage.ChangeSet[hookstage.BidderRequestPayload]

	err := updateBType(cfg, payload, &blockingAttrs, &result, &changeSet)
	require.NoError(t, err)

	// Apply mutations
	mutations := changeSet.Mutations()
	for _, mutation := range mutations {
		updatedPayload, err := mutation.Apply(payload)
		require.NoError(t, err)
		payload = updatedPayload
	}

	// Check that Banner was NOT created for video-only impression
	for _, imp := range payload.Request.BidRequest.Imp {
		switch imp.ID {
		case "imp-video-only":
			assert.NotNil(t, imp.Video, "Video should exist")
			assert.Nil(t, imp.Banner, "Banner should NOT be created")
		case "imp-with-banner":
			assert.NotNil(t, imp.Banner, "Banner should exist")
			assert.Len(t, imp.Banner.BType, 3, "Banner should have btype applied")
		}
	}
}

func TestCreateBAttrMutation_EmptyValues(t *testing.T) {
	bAttrByImp := map[string][]int{
		"imp1": {},
		"imp2": {1, 2},
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

	// Check that empty values are handled correctly
	for _, imp := range updatedPayload.Request.BidRequest.Imp {
		switch imp.ID {
		case "imp1":
			require.NotNil(t, imp.Banner)
			// Empty array should result in empty BAttr
			assert.Len(t, imp.Banner.BAttr, 0)
		case "imp2":
			require.NotNil(t, imp.Banner)
			assert.Len(t, imp.Banner.BAttr, 2)
		}
	}
}

func TestCreateBTypeMutation_EmptyValues(t *testing.T) {
	bTypeByImp := map[string][]int{
		"imp1": {},
		"imp2": {1, 2},
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

	mutation := createBTypeMutation(bTypeByImp)
	updatedPayload, err := mutation(payload)
	require.NoError(t, err)

	for _, imp := range updatedPayload.Request.BidRequest.Imp {
		switch imp.ID {
		case "imp1":
			require.NotNil(t, imp.Banner)
			assert.Len(t, imp.Banner.BType, 0)
		case "imp2":
			require.NotNil(t, imp.Banner)
			assert.Len(t, imp.Banner.BType, 2)
		}
	}
}

func TestMediaTypesFrom(t *testing.T) {
	request := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:     "imp1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			{
				ID:    "imp2",
				Audio: &openrtb2.Audio{},
			},
		},
	}

	mediaTypes := mediaTypesFrom(request)

	assert.Len(t, mediaTypes, 3)
	assert.Contains(t, mediaTypes, "banner")
	assert.Contains(t, mediaTypes, "video")
	assert.Contains(t, mediaTypes, "audio")
	assert.NotContains(t, mediaTypes, "native")
}

func TestMediaTypesFromImp(t *testing.T) {
	tests := []struct {
		name     string
		imp      openrtb2.Imp
		expected []string
	}{
		{
			name: "banner only",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
			},
			expected: []string{"banner"},
		},
		{
			name: "video only",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{},
			},
			expected: []string{"video"},
		},
		{
			name: "audio only",
			imp: openrtb2.Imp{
				Audio: &openrtb2.Audio{},
			},
			expected: []string{"audio"},
		},
		{
			name: "native only",
			imp: openrtb2.Imp{
				Native: &openrtb2.Native{},
			},
			expected: []string{"native"},
		},
		{
			name: "multiple types",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
				Audio:  &openrtb2.Audio{},
			},
			expected: []string{"banner", "video", "audio"},
		},
		{
			name:     "no media types",
			imp:      openrtb2.Imp{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mediaTypesFromImp(tt.imp)
			assert.Len(t, result, len(tt.expected))
			for _, mt := range tt.expected {
				assert.Contains(t, result, mt)
			}
		})
	}
}

func TestMediaTypes_String(t *testing.T) {
	tests := []struct {
		name     string
		mt       mediaTypes
		expected string
	}{
		{
			name: "multiple types sorted",
			mt: mediaTypes{
				"video":  struct{}{},
				"banner": struct{}{},
				"audio":  struct{}{},
			},
			expected: "audio, banner, video",
		},
		{
			name:     "empty",
			mt:       mediaTypes{},
			expected: "",
		},
		{
			name: "all types",
			mt: mediaTypes{
				"audio":  struct{}{},
				"banner": struct{}{},
				"native": struct{}{},
				"video":  struct{}{},
			},
			expected: "audio, banner, native, video",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mt.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMediaTypes_Intersects(t *testing.T) {
	mt := mediaTypes{
		"banner": struct{}{},
		"video":  struct{}{},
	}

	tests := []struct {
		name     string
		input    []string
		expected bool
	}{
		{
			name:     "exact match",
			input:    []string{"banner"},
			expected: true,
		},
		{
			name:     "case insensitive match",
			input:    []string{"BANNER"},
			expected: true,
		},
		{
			name:     "multiple with match",
			input:    []string{"audio", "video"},
			expected: true,
		},
		{
			name:     "no match",
			input:    []string{"audio", "native"},
			expected: false,
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mt.intersects(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Conditions
		wantErr   bool
	}{
		{
			name: "valid - bidders only",
			condition: Conditions{
				Bidders: []string{"bidder1"},
			},
			wantErr: false,
		},
		{
			name: "valid - media types only",
			condition: Conditions{
				MediaTypes: []string{"banner"},
			},
			wantErr: false,
		},
		{
			name: "valid - both present",
			condition: Conditions{
				Bidders:    []string{"bidder1"},
				MediaTypes: []string{"banner"},
			},
			wantErr: false,
		},
		{
			name:      "invalid - both absent",
			condition: Conditions{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCondition(tt.condition)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterByMediaType_NoMatches(t *testing.T) {
	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						Video: &openrtb2.Video{},
					},
				},
			},
		},
	}

	overrides := map[string][]int{
		"imp1": {1, 2, 3},
	}

	// Filter for Banner, but impression has only Video
	filtered := filterByMediaType(payload, overrides, func(imp openrtb2.Imp) bool {
		return imp.Banner != nil
	})

	assert.Len(t, filtered, 0, "Should not include impressions without matching media type")
}

func TestCreateBAttrMutation_VideoMediaType(t *testing.T) {
	bAttrByImp := map[string][]int{
		"imp1": {1, 2},
	}

	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						Video: &openrtb2.Video{},
					},
				},
			},
		},
	}

	mutation := createBAttrMutation(bAttrByImp, "video")
	updatedPayload, err := mutation(payload)
	require.NoError(t, err)

	imp := updatedPayload.Request.BidRequest.Imp[0]
	require.NotNil(t, imp.Video)
	assert.Len(t, imp.Video.BAttr, 2)
	assert.Equal(t, adcom1.CreativeAttribute(1), imp.Video.BAttr[0])
	assert.Equal(t, adcom1.CreativeAttribute(2), imp.Video.BAttr[1])
}

func TestCreateBAttrMutation_AudioMediaType(t *testing.T) {
	bAttrByImp := map[string][]int{
		"imp1": {7, 8, 9},
	}

	payload := hookstage.BidderRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						Audio: &openrtb2.Audio{},
					},
				},
			},
		},
	}

	mutation := createBAttrMutation(bAttrByImp, "audio")
	updatedPayload, err := mutation(payload)
	require.NoError(t, err)

	imp := updatedPayload.Request.BidRequest.Imp[0]
	require.NotNil(t, imp.Audio)
	assert.Len(t, imp.Audio.BAttr, 3)
	assert.Equal(t, adcom1.CreativeAttribute(7), imp.Audio.BAttr[0])
	assert.Equal(t, adcom1.CreativeAttribute(8), imp.Audio.BAttr[1])
	assert.Equal(t, adcom1.CreativeAttribute(9), imp.Audio.BAttr[2])
}
