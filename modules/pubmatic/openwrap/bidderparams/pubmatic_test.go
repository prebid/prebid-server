package bidderparams

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache"
	mock_cache "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func getTestImp(tagID string, banner bool, video bool) openrtb2.Imp {
	if banner {
		return openrtb2.Imp{
			ID: "111",
			Banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](200),
				H: ptrutil.ToPtr[int64](300),
				Format: []openrtb2.Format{
					{
						W: 400,
						H: 500,
					},
				},
			},
			TagID: tagID,
		}
	} else if video {
		return openrtb2.Imp{
			ID: "111",
			Video: &openrtb2.Video{
				W: ptrutil.ToPtr[int64](200),
				H: ptrutil.ToPtr[int64](300),
			},
			TagID: tagID,
		}
	}

	return openrtb2.Imp{
		ID: "111",
		Native: &openrtb2.Native{
			Request: "test",
			Ver:     "testVer",
		},
		TagID: tagID,
	}
}

func TestGetImpExtPubMaticKeyWords(t *testing.T) {
	type args struct {
		impExt     models.ImpExtension
		bidderCode string
	}
	tests := []struct {
		name string
		args args
		want []*openrtb_ext.ExtImpPubmaticKeyVal
	}{
		{
			name: "empty_impExt_bidder",
			args: args{
				impExt: models.ImpExtension{
					Bidder: nil,
				},
			},
			want: nil,
		},
		{
			name: "bidder_code_is_not_present_in_impExt_bidder",
			args: args{
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"appnexus": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
				},
				bidderCode: "pubmatic",
			},
			want: nil,
		},
		{
			name: "impExt_bidder_contains_key_value_pair_for_bidder_code",
			args: args{
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
				},
				bidderCode: "pubmatic",
			},
			want: []*openrtb_ext.ExtImpPubmaticKeyVal{
				{
					Key:    "test_key1",
					Values: []string{"test_value1", "test_value2"},
				},
				{
					Key:    "test_key2",
					Values: []string{"test_value1", "test_value2"},
				},
			},
		},
		{
			name: "impExt_bidder_contains_key_value_pair_for_bidder_code_ignore_key_value_pair_with_no_values",
			args: args{
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{},
								},
							},
						},
					},
				},
				bidderCode: "pubmatic",
			},
			want: []*openrtb_ext.ExtImpPubmaticKeyVal{
				{
					Key:    "test_key1",
					Values: []string{"test_value1", "test_value2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getImpExtPubMaticKeyWords(tt.args.impExt, tt.args.bidderCode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDealTier(t *testing.T) {
	type args struct {
		impExt     models.ImpExtension
		bidderCode string
	}
	tests := []struct {
		name string
		args args
		want *openrtb_ext.DealTier
	}{
		{
			name: "impExt_bidder_is_empty",
			args: args{
				impExt: models.ImpExtension{
					Bidder: nil,
				},
			},
			want: nil,
		},
		{
			name: "bidder_code_is_not_present_in_impExt_bidder",
			args: args{
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"appnexus": {
							DealTier: &openrtb_ext.DealTier{
								Prefix:      "test",
								MinDealTier: 10,
							},
						},
					},
				},
				bidderCode: "pubmatic",
			},
			want: nil,
		},
		{
			name: "bidder_code_is_present_in_impExt_bidder",
			args: args{
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							DealTier: &openrtb_ext.DealTier{
								Prefix:      "test",
								MinDealTier: 10,
							},
						},
					},
				},
				bidderCode: "pubmatic",
			},
			want: &openrtb_ext.DealTier{
				Prefix:      "test",
				MinDealTier: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDealTier(tt.args.impExt, tt.args.bidderCode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPreparePubMaticParamsV25(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)

	type args struct {
		rctx       models.RequestCtx
		cache      cache.Cache
		bidRequest openrtb2.BidRequest
		imp        openrtb2.Imp
		impExt     models.ImpExtension
		partnerID  int
	}
	type want struct {
		matchedSlot    string
		matchedPattern string
		isRegexSlot    bool
		params         []byte
		wantErr        bool
	}
	tests := []struct {
		name  string
		args  args
		setup func()
		want  want
	}{
		{
			name: "testRequest",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit12345@div1@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "/Test_Adunit1234@Div1@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"/test_adunit1234@div1@200x300"},
					HashValueMap: map[string]string{
						"/test_adunit1234@div1@200x300": "2aa34b52a9e941c1594af7565e599c8d",
					},
				})
				mockCache.EXPECT().Get("psregex_5890_123_1_1_/Test_Adunit1234@Div1@200x300").Return(nil, false)
				mockCache.EXPECT().Set("psregex_5890_123_1_1_/Test_Adunit1234@Div1@200x300", regexSlotEntry{SlotName: "/Test_Adunit1234@Div1@200x300", RegexPattern: "/test_adunit1234@div1@200x300"}).Times(1)
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: "/test_adunit1234@div1@200x300",
				isRegexSlot:    true,
				params:         []byte(`{"publisherId":"5890","adSlot":"2aa34b52a9e941c1594af7565e599c8d","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "exact_matched_slot_found_adslot_updated_from_PubMatic_secondary_flow",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic2",
							models.BidderCode:          "pubmatic2",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
							models.KEY_PROFILE_ID:      "1323",
							models.KEY_PUBLISHER_ID:    "301",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic2": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@div1@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "/Test_Adunit1234@DIV1@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"301","adSlot":"/Test_Adunit1234@DIV1@200x300","wrapper":{"profile":1323},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "exact_matched_slot_found_adslot_updated_from_PubMatic_secondary_flow_for_prebids2s_regex",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic2",
							models.BidderCode:          "pubmatic2",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_RE_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
							models.KEY_PROFILE_ID:      "1323",
							models.KEY_PUBLISHER_ID:    "301",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic2": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "/Test_Adunit1234@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"301","adSlot":"/Test_Adunit1234@200x300","wrapper":{"profile":1323},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "exact_matched_slot_found_adslot_updated_from_PubMatic_alias_flow_for_prebids2s_regex",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic_alias",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_RE_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
							models.IsAlias:             "1",
							models.PubID:               "301",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic_alias": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "/Test_Adunit1234@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"301","adSlot":"/Test_Adunit1234@200x300","keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "exact_matched_slot_found_adSlot_upadted_from_owSlotName_prebids2s_regex",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_RE_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@200x300",
						SlotMappings: map[string]interface{}{
							"site":                  "12313",
							"adtag":                 "45343",
							models.KEY_OW_SLOT_NAME: "/Test_Adunit1234@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@200x300","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "prebids2s_regex_matched_slot_found_adSlot_upadted_from_hashValue",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_RE_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					".*@.*": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@200x300",
						SlotMappings: map[string]interface{}{
							"site":  "12313",
							"adtag": "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
					HashValueMap: map[string]string{
						".*@.*": "2aa34b52a9e941c1594af7565e599c8d",
					},
				})
				mockCache.EXPECT().Get("psregex_5890_123_1_1_/Test_Adunit1234@200x300").Return(regexSlotEntry{
					SlotName:     "/Test_Adunit1234@200x300",
					RegexPattern: ".*@.*",
				}, true)
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@200x300",
				matchedPattern: ".*@.*",
				isRegexSlot:    true,
				params:         []byte(`{"publisherId":"5890","adSlot":"2aa34b52a9e941c1594af7565e599c8d","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "exact_matched_slot_found_adSlot_upadted_from_owSlotName",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@div1@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":                  "12313",
							"adtag":                 "45343",
							models.KEY_OW_SLOT_NAME: "/Test_Adunit1234@DIV1@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@DIV1@200x300","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "regex_matched_slot_found_adSlot_upadted_from_hashValue",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					".*@div.*@.*": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":  "12313",
							"adtag": "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
					HashValueMap: map[string]string{
						".*@Div.*@.*": "2aa34b52a9e941c1594af7565e599c8d",
					},
				})
				mockCache.EXPECT().Get("psregex_5890_123_1_1_/Test_Adunit1234@Div1@200x300").Return(regexSlotEntry{
					SlotName:     "/Test_Adunit1234@Div1@200x300",
					RegexPattern: ".*@Div.*@.*",
				}, true)
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: ".*@Div.*@.*",
				isRegexSlot:    true,
				params:         []byte(`{"publisherId":"5890","adSlot":"2aa34b52a9e941c1594af7565e599c8d","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "regex_matched_slot_found_adSlot_upadted_from_hashValue_for_test_value_1",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					".*@div.*@.*": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":  "12313",
							"adtag": "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
					HashValueMap: map[string]string{
						".*@Div.*@.*": "2aa34b52a9e941c1594af7565e599c8d",
					},
				})
				mockCache.EXPECT().Get("psregex_5890_123_1_1_/Test_Adunit1234@Div1@200x300").Return(regexSlotEntry{
					SlotName:     "/Test_Adunit1234@Div1@200x300",
					RegexPattern: ".*@Div.*@.*",
				}, true)
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: ".*@Div.*@.*",
				isRegexSlot:    true,
				params:         []byte(`{"publisherId":"5890","adSlot":"2aa34b52a9e941c1594af7565e599c8d","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "valid_pubmatic_native_params",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", false, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@1x1": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@1x1",
						SlotMappings: map[string]interface{}{
							"site":                  "12313",
							"adtag":                 "45343",
							models.KEY_OW_SLOT_NAME: "/Test_Adunit1234@1x1",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@1x1",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@1x1","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "valid_pubmatic_video_params",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", false, true),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@0x0": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@0x0",
						SlotMappings: map[string]interface{}{
							"site":                  "12313",
							"adtag":                 "45343",
							models.KEY_OW_SLOT_NAME: "/Test_Adunit1234@0x0",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@0x0",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@0x0","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "pubmatic_param_for_native_default",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", false, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"random": {
						SlotName: "/Test_Adunit1234",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"random"},
					HashValueMap: map[string]string{
						"random": "2aa34b52a9e941c1594af7565e599c8d",
					},
				})
			},
			want: want{
				matchedSlot:    "",
				matchedPattern: "/Test_Adunit1234@1x1",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "pubmatic_param_for_banner_default",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"random": {
						SlotName: "/Test_Adunit1234",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"random"},
					HashValueMap: map[string]string{
						"random": "2aa34b52a9e941c1594af7565e599c8d",
					},
				})
			},
			want: want{
				matchedSlot:    "",
				matchedPattern: "/Test_Adunit1234@200x300",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "pubmatic_param_for_video_default",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", false, true),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"random": {
						SlotName: "/Test_Adunit1234",
						SlotMappings: map[string]interface{}{
							models.SITE_CACHE_KEY: "12313",
							models.TAG_CACHE_KEY:  "45343",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"random"},
					HashValueMap: map[string]string{
						"random": "2aa34b52a9e941c1594af7565e599c8d",
					},
				})
			},
			want: want{
				matchedSlot:    "",
				matchedPattern: "/Test_Adunit1234@0x0",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "For_test_value_1_for_regex",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit12345@div1@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "/Test_Adunit1234@DIV1@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"*", ".*@.*@.*"},
					HashValueMap: map[string]string{
						".*@.*@.*": "2aa34b52a9e941c1594af7565e599c8d", // Code should match the given slot name with this regex
					},
				})
				mockCache.EXPECT().Get("psregex_5890_123_1_1_/Test_Adunit1234@Div1@200x300").Return(nil, false)
				mockCache.EXPECT().Set("psregex_5890_123_1_1_/Test_Adunit1234@Div1@200x300", regexSlotEntry{SlotName: "/Test_Adunit1234@Div1@200x300", RegexPattern: ".*@.*@.*"}).Times(1)
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: ".*@.*@.*",
				isRegexSlot:    true,
				params:         []byte(`{"publisherId":"5890","adSlot":"2aa34b52a9e941c1594af7565e599c8d","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "For_test_value_1_for_non_regex",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", false, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@1x1": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@1x1",
						SlotMappings: map[string]interface{}{
							"site":                  "12313",
							"adtag":                 "45343",
							models.KEY_OW_SLOT_NAME: "/Test_Adunit1234@1x1",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@1x1",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@1x1","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "For_test_value_1_exact_matched_slot_found_adslot_updated_from_PubMatic_secondary_flow",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic2",
							models.BidderCode:          "pubmatic2",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
							models.KEY_PROFILE_ID:      "1323",
							models.KEY_PUBLISHER_ID:    "301",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic2": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@div1@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "/Test_Adunit1234@DIV1@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"301","adSlot":"/Test_Adunit1234@DIV1@200x300","wrapper":{"profile":1323},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "For_test_value_1_exact_matched_slot_found_adslot_updated_from_PubMatic_secondary_flow_for_different_slotname",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 1,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic2",
							models.BidderCode:          "pubmatic2",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
							models.KEY_PROFILE_ID:      "1323",
							models.KEY_PUBLISHER_ID:    "301",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic2": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@div1@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "pubmatic2-slot",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"301","adSlot":"pubmatic2-slot","wrapper":{"profile":1323},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "For_test_value_2_with_regex",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 2,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@Div1@200x300","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "For_test_value_2_with_non_regex",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 2,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					DisplayID:     1,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@200x300","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}]}`),
				wantErr:        false,
			},
		},
		{
			name: "exact_matched_slot_found_adslot_and_applovin_floors_updated",
			args: args{
				rctx: models.RequestCtx{
					IsTestRequest: 0,
					PubID:         5890,
					PubIDStr:      "5890",
					ProfileID:     123,
					ProfileIDStr:  "123",
					DisplayID:     1,
					Endpoint:      models.EndpointAppLovinMax,
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.TIMEOUT:             "200",
							models.KEY_GEN_PATTERN:     "_AU_@_DIV_@_W_x_H_",
							models.SERVER_SIDE_FLAG:    "1",
						},
					},
					MultiFloors: map[string]*models.MultiFloors{
						"111": {Tier1: 1.5, Tier2: 1.2, Tier3: 2.2},
					},
				},
				cache: mockCache,
				impExt: models.ImpExtension{
					Bidder: map[string]*models.BidderExtension{
						"pubmatic": {
							KeyWords: []models.KeyVal{
								{
									Key:    "test_key1",
									Values: []string{"test_value1", "test_value2"},
								},
								{
									Key:    "test_key2",
									Values: []string{"test_value1", "test_value2"},
								},
							},
						},
					},
					Wrapper: &models.ExtImpWrapper{
						Div: "Div1",
					},
					OWSDK: map[string]any{"ctaoverlay": 1},
				},
				imp:       getTestImp("/Test_Adunit1234", true, false),
				partnerID: 1,
			},
			setup: func() {
				mockCache.EXPECT().GetMappingsFromCacheV25(gomock.Any(), gomock.Any()).Return(map[string]models.SlotMapping{
					"/test_adunit1234@div1@200x300": {
						PartnerId: 1,
						AdapterId: 1,
						SlotName:  "/Test_Adunit1234@Div1@200x300",
						SlotMappings: map[string]interface{}{
							"site":     "12313",
							"adtag":    "45343",
							"slotName": "/Test_Adunit1234@DIV1@200x300",
						},
					},
				})
				mockCache.EXPECT().GetSlotToHashValueMapFromCacheV25(gomock.Any(), gomock.Any()).Return(models.SlotMappingInfo{
					OrderedSlotList: []string{"test", "test1"},
				})
			},
			want: want{
				matchedSlot:    "/Test_Adunit1234@Div1@200x300",
				matchedPattern: "",
				isRegexSlot:    false,
				params:         []byte(`{"publisherId":"5890","adSlot":"/Test_Adunit1234@DIV1@200x300","wrapper":{"version":1,"profile":123},"keywords":[{"key":"test_key1","value":["test_value1","test_value2"]},{"key":"test_key2","value":["test_value1","test_value2"]}],"floors":[1.5,1.2,2.2],"owsdk":{"ctaoverlay":1}}`),
				wantErr:        false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			matchedSlot, matchedPattern, isRegexSlot, params, err := PreparePubMaticParamsV25(tt.args.rctx, tt.args.cache, tt.args.bidRequest, tt.args.imp, tt.args.impExt, tt.args.partnerID)
			if (err != nil) != tt.want.wantErr {
				assert.Equal(t, tt.want.wantErr, err != nil)
				return
			}
			assert.Equal(t, tt.want.matchedSlot, matchedSlot)
			assert.Equal(t, tt.want.matchedPattern, matchedPattern)
			assert.Equal(t, tt.want.isRegexSlot, isRegexSlot)
			assert.Equal(t, tt.want.params, params)
		})
	}
}

func createSlotMapping(slotName string, mappings map[string]interface{}) models.SlotMapping {
	return models.SlotMapping{
		PartnerId:    0,
		AdapterId:    0,
		VersionId:    0,
		SlotName:     slotName,
		SlotMappings: mappings,
		Hash:         "",
		OrderID:      0,
	}
}

func TestGetMatchingSlotAndPattern(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)

	type args struct {
		rctx            models.RequestCtx
		cache           cache.Cache
		slots           []string
		slotMap         map[string]models.SlotMapping
		slotMappingInfo models.SlotMappingInfo
		isRegexKGP      bool
		isRegexSlot     bool
		partnerID       int
		extImpPubMatic  *openrtb_ext.ExtImpPubmatic
		imp             openrtb2.Imp
	}
	type want struct {
		matchedSlot    string
		matchedPattern string
		isRegexSlot    bool
	}
	tests := []struct {
		name  string
		args  args
		setup func()
		want  want
	}{
		{
			name: "found_matced_regex_slot",
			args: args{
				rctx: models.RequestCtx{
					PubID:     5890,
					ProfileID: 123,
					DisplayID: 1,
				},
				partnerID: 1,
				slots:     []string{"AU123@Div1@728x90"},
				slotMappingInfo: models.SlotMappingInfo{
					OrderedSlotList: []string{"*", ".*@.*@.*"},
					HashValueMap: map[string]string{
						".*@.*@.*": "2aa34b52a9e941c1594af7565e599c8d", // Code should match the given slot name with this regex
					},
				},
				slotMap: map[string]models.SlotMapping{
					"AU123@Div1@728x90": {
						SlotMappings: map[string]interface{}{
							"site":  "123123",
							"adtag": "45343",
						},
					},
				},
				cache:          mockCache,
				isRegexKGP:     true,
				isRegexSlot:    false,
				extImpPubMatic: &openrtb_ext.ExtImpPubmatic{},
				imp:            openrtb2.Imp{},
			},
			setup: func() {
				mockCache.EXPECT().Get("psregex_5890_123_1_1_AU123@Div1@728x90").Return(nil, false)
				mockCache.EXPECT().Set("psregex_5890_123_1_1_AU123@Div1@728x90", regexSlotEntry{SlotName: "AU123@Div1@728x90", RegexPattern: ".*@.*@.*"}).Times(1)
			},
			want: want{
				matchedSlot:    "AU123@Div1@728x90",
				matchedPattern: ".*@.*@.*",
				isRegexSlot:    true,
			},
		},
		{
			name: "not_found_matced_regex_slot",
			args: args{
				rctx: models.RequestCtx{
					PubID:     5890,
					ProfileID: 123,
					DisplayID: 1,
				},
				partnerID: 1,
				slots:     []string{"AU123@Div1@728x90"},
				slotMap: map[string]models.SlotMapping{
					"AU123@Div1@728x90": {
						SlotMappings: map[string]interface{}{
							"site":  "123123",
							"adtag": "45343",
						},
					},
				},
				cache:          mockCache,
				isRegexKGP:     true,
				isRegexSlot:    false,
				extImpPubMatic: &openrtb_ext.ExtImpPubmatic{},
				imp:            openrtb2.Imp{},
			},
			setup: func() {
				mockCache.EXPECT().Get("psregex_5890_123_1_1_AU123@Div1@728x90").Return(nil, false)
			},
			want: want{
				matchedSlot:    "",
				matchedPattern: "",
				isRegexSlot:    false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			matchedSlot, matchedPattern, isRegexSlot := getMatchingSlotAndPattern(tt.args.rctx, tt.args.cache, tt.args.slots, tt.args.slotMap, tt.args.slotMappingInfo, tt.args.isRegexKGP, tt.args.isRegexSlot, tt.args.partnerID, tt.args.extImpPubMatic, tt.args.imp)
			assert.Equal(t, tt.want.matchedSlot, matchedSlot)
			assert.Equal(t, tt.want.matchedPattern, matchedPattern)
			assert.Equal(t, tt.want.isRegexSlot, isRegexSlot)
		})
	}
}

func TestGetPubMaticPublisherID(t *testing.T) {
	type args struct {
		rctx      models.RequestCtx
		partnerID int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "pubmatic partner",
			args: args{
				partnerID: 789,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						789: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.PubID:               "5890",
						},
					},
					PubID:    5890,
					PubIDStr: "5890",
				},
			},
			want: "5890",
		},
		{
			name: "pubmatic secondary partner",
			args: args{
				partnerID: 123,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						123: {
							models.PREBID_PARTNER_NAME: "pubmatic2",
							models.BidderCode:          "pubmatic2",
							models.KEY_PUBLISHER_ID:    "301",
							models.PubID:               "5890",
						},
					},
					PubID:    5890,
					PubIDStr: "5890",
				},
			},
			want: "301",
		},
		{
			name: "pubmatic alias partner",
			args: args{
				partnerID: 456,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						456: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubm_alias",
							models.PubID:               "301",
							models.IsAlias:             "1",
						},
					},
					PubID:    5890,
					PubIDStr: "5890",
				},
			},
			want: "301",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPubMaticPublisherID(tt.args.rctx, tt.args.partnerID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPubMaticWrapperExt(t *testing.T) {
	type args struct {
		rctx      models.RequestCtx
		partnerID int
	}
	tests := []struct {
		name string
		args args
		want json.RawMessage
	}{
		{
			name: "pubmatic partner",
			args: args{
				partnerID: 789,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						789: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubmatic",
							models.PubID:               "5890",
						},
					},
					DisplayID: 1,
					ProfileID: 1234,
					PubID:     5890,
				},
			},
			want: json.RawMessage(`{"version":1,"profile":1234}`),
		},
		{
			name: "pubmatic secondary partner",
			args: args{
				partnerID: 123,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						123: {
							models.PREBID_PARTNER_NAME: "pubmatic2",
							models.BidderCode:          "pubmatic2",
							models.KEY_PROFILE_ID:      "222",
						},
					},
					DisplayID: 1,
					ProfileID: 1234,
				},
			},
			want: json.RawMessage(`{"profile":222}`),
		},
		{
			name: "pubmatic alias partner with pubID different from incoming request pubID",
			args: args{
				partnerID: 456,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						456: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubm_alias",
							models.PubID:               "301",
							models.IsAlias:             "1",
						},
					},
					DisplayID: 1,
					ProfileID: 1234,
					PubID:     5890,
					PubIDStr:  "5890",
				},
			},
			want: nil,
		},
		{
			name: "pubmatic alias partner with pubID same as incoming request pubID",
			args: args{
				partnerID: 456,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						456: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubm_alias",
							models.PubID:               "5890",
							models.IsAlias:             "1",
						},
					},
					DisplayID: 1,
					ProfileID: 1234,
					PubID:     5890,
					PubIDStr:  "5890",
				},
			},
			want: json.RawMessage(`{"version":1,"profile":1234}`),
		},
		{
			name: "pubmatic alias partner with no pubID in partner config",
			args: args{
				partnerID: 456,
				rctx: models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						456: {
							models.PREBID_PARTNER_NAME: "pubmatic",
							models.BidderCode:          "pubm_alias",
							models.IsAlias:             "1",
						},
					},
					DisplayID: 1,
					ProfileID: 1234,
					PubID:     5890,
					PubIDStr:  "5890",
				},
			},
			want: json.RawMessage(`{"version":1,"profile":1234}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPubMaticWrapperExt(tt.args.rctx, tt.args.partnerID)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			} else {
				assert.JSONEq(t, string(tt.want), string(got))
			}
		})
	}
}
