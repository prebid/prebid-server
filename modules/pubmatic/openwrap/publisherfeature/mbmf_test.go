package publisherfeature

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache"
	mock_cache "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/stretchr/testify/assert"
)

func TestFeatureUpdateMBMF(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)

	type fields struct {
		cache            cache.Cache
		publisherFeature map[int]map[int]models.FeatureData
		mbmf             *mbmf
	}
	tests := []struct {
		name   string
		fields fields
		setup  func()
		want   *mbmf
	}{
		{
			name: "publisherFeature_map_is_nil",
			fields: fields{
				cache:            nil,
				publisherFeature: nil,
				mbmf:             newMBMF(),
			},
			setup: func() {},
			want: func() *mbmf {
				m := mbmf{
					data: [2]mbmfData{
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
					},
				}
				m.index.Store(0)
				return &m
			}(),
		},
		{
			name: "publisherFeature_map_is_empty",
			fields: fields{
				cache:            mockCache,
				publisherFeature: map[int]map[int]models.FeatureData{},
				mbmf:             newMBMF(),
			},
			setup: func() {
				mockCache.EXPECT().GetProfileAdUnitMultiFloors().Return(models.ProfileAdUnitMultiFloors{}, nil)
			},
			want: func() *mbmf {
				m := mbmf{
					data: [2]mbmfData{
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
					},
				}
				m.index.Store(1)
				return &m
			}(),
		},
		{
			name: "publisherFeature_map_contain_mbmf_country",
			fields: fields{
				cache: mockCache,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   `US,DE`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			setup: func() {
				mockCache.EXPECT().GetProfileAdUnitMultiFloors().Return(models.ProfileAdUnitMultiFloors{}, nil)
			},
			want: func() *mbmf {
				m := mbmf{
					data: [2]mbmfData{
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         map[int]models.HashSet{123: {"US": {}, "DE": {}}},
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
					},
				}
				m.index.Store(1)
				return &m
			}(),
		},
		{
			name: "publisherFeature_map_contain_mbmf_instl_floors",
			fields: fields{
				cache: mockCache,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   `US,DE`,
						},
					},
					5890: {
						models.FeatureMBMFInstlFloors: {
							Enabled: 1,
							Value:   `{"tier1":1.0,"tier2":2.0,"tier3":3.0}`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			setup: func() {
				mockCache.EXPECT().GetProfileAdUnitMultiFloors().Return(models.ProfileAdUnitMultiFloors{}, nil)
			},
			want: func() *mbmf {
				m := mbmf{
					data: [2]mbmfData{
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         map[int]models.HashSet{123: {"US": {}, "DE": {}}},
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              map[int]*models.MultiFloors{5890: {IsActive: true, Tier1: 1.0, Tier2: 2.0, Tier3: 3.0}},
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
					},
				}
				m.index.Store(1)
				return &m
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				cache:            tt.fields.cache,
				publisherFeature: tt.fields.publisherFeature,
				mbmf:             tt.fields.mbmf,
			}
			tt.setup()
			f.updateMBMF()
			assert.Equal(t, tt.want, f.mbmf)
		})
	}
}

func TestFeatureUpdateMBMFCountries(t *testing.T) {
	type fields struct {
		cache            cache.Cache
		publisherFeature map[int]map[int]models.FeatureData
		mbmf             *mbmf
	}
	tests := []struct {
		name                     string
		fields                   fields
		wantMBMFEnabledCountries map[int]models.HashSet
	}{
		{
			name: "publisherFeature_map_is_present_but_mbmf_is_not_present_in_DB",
			fields: fields{
				cache:            nil,
				publisherFeature: map[int]map[int]models.FeatureData{},
				mbmf:             newMBMF(),
			},
			wantMBMFEnabledCountries: make(map[int]models.HashSet),
		},
		{
			name: "mbmf_enabled_countries",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   `US,DE`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			wantMBMFEnabledCountries: map[int]models.HashSet{
				123: {
					"US": struct{}{},
					"DE": struct{}{},
				},
			},
		},
		{
			name: "mbmf_enabled_countries_with_multiple_publishers",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   `US,DE`,
						},
					},
					456: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   `IN,FR`,
						},
					},
					0: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   `JP,KR`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			wantMBMFEnabledCountries: map[int]models.HashSet{
				123: {
					"US": {},
					"DE": {},
				},
				456: {
					"IN": {},
					"FR": {},
				},
				0: {
					"JP": {},
					"KR": {},
				},
			},
		},
		{
			name: "mbmf_enabled_countries_with_space_in_value",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					0: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   `  US,DE  `,
						},
					},
				},
				mbmf: newMBMF(),
			},
			wantMBMFEnabledCountries: map[int]models.HashSet{
				0: {
					"US": {},
					"DE": {},
				},
			},
		},
		{
			name: "mbmf_enabled_countries_with_feature_disabled",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					0: {
						models.FeatureMBMFCountry: {
							Enabled: 0,
							Value:   `US,DE`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			wantMBMFEnabledCountries: map[int]models.HashSet{},
		},
		{
			name: "mbmf_enabled_countries_with_feature_enabled_but_empty_value",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					0: {
						models.FeatureMBMFCountry: {
							Enabled: 1,
							Value:   ``,
						},
					},
				},
				mbmf: newMBMF(),
			},
			wantMBMFEnabledCountries: map[int]models.HashSet{0: {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				publisherFeature: tt.fields.publisherFeature,
				mbmf:             tt.fields.mbmf,
			}
			defer func() {
				f = nil
			}()
			f.updateMBMFCountries(0)
			assert.Equal(t, tt.wantMBMFEnabledCountries, f.mbmf.data[0].enabledCountries)
		})
	}
}

func TestFeatureUpdateMBMFPublishers(t *testing.T) {
	type fields struct {
		cache            cache.Cache
		publisherFeature map[int]map[int]models.FeatureData
		mbmf             *mbmf
	}
	tests := []struct {
		name   string
		fields fields
		want   map[int]bool
	}{
		{
			name: "empty publisherFeature map",
			fields: fields{
				cache:            nil,
				publisherFeature: map[int]map[int]models.FeatureData{},
				mbmf:             newMBMF(),
			},
			want: make(map[int]bool),
		},
		{
			name: "publisher feature enabled and disabled",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFPublisher: {
							Enabled: 1,
						},
					},
					456: {
						models.FeatureMBMFPublisher: {
							Enabled: 0,
						},
					},
				},
				mbmf: newMBMF(),
			},
			want: map[int]bool{
				123: true,
				456: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				publisherFeature: tt.fields.publisherFeature,
				mbmf:             tt.fields.mbmf,
			}
			// Get current index
			currIdx := f.mbmf.index.Load()
			// Update using next index (currIdx ^ 1)
			f.updateMBMFPublishers(currIdx ^ 1)
			// Store the next index
			f.mbmf.index.Store(currIdx ^ 1)
			// Assert using the new index
			assert.Equal(t, tt.want, f.mbmf.data[currIdx^1].enabledPublishers)
		})
	}
}

func TestFeatureUpdateProfileAdUnitLevelFloors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)

	type fields struct {
		cache cache.Cache
		mbmf  *mbmf
	}
	tests := []struct {
		name               string
		fields             fields
		setup              func()
		expectedMBMFFloors models.ProfileAdUnitMultiFloors
	}{
		{
			name: "query failed",
			fields: fields{
				cache: mockCache,
				mbmf:  newMBMF(),
			},
			setup: func() {
				mockCache.EXPECT().GetProfileAdUnitMultiFloors().Return(nil, errors.New("QUERY FAILED"))
			},
			expectedMBMFFloors: make(models.ProfileAdUnitMultiFloors),
		},
		{
			name: "query success",
			fields: fields{
				cache: mockCache,
				mbmf:  newMBMF(),
			},
			setup: func() {
				mockCache.EXPECT().GetProfileAdUnitMultiFloors().Return(models.ProfileAdUnitMultiFloors{
					123: {
						"adunit1": &models.MultiFloors{
							IsActive: true,
							Tier1:    1.0,
							Tier2:    2.0,
							Tier3:    3.0,
						},
					},
				}, nil)
			},
			expectedMBMFFloors: models.ProfileAdUnitMultiFloors{
				123: {
					"adunit1": &models.MultiFloors{
						IsActive: true,
						Tier1:    1.0,
						Tier2:    2.0,
						Tier3:    3.0,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			f := &feature{
				cache: tt.fields.cache,
				mbmf:  tt.fields.mbmf,
			}
			defer func() {
				f = nil
			}()
			// Get current index
			currIdx := f.mbmf.index.Load()
			// Update using next index (currIdx ^ 1)
			f.updateProfileAdUnitLevelFloors(currIdx ^ 1)
			// Store the next index
			f.mbmf.index.Store(currIdx ^ 1)
			// Assert using the new index
			assert.Equal(t, tt.expectedMBMFFloors, f.mbmf.data[currIdx^1].profileAdUnitLevelFloors, tt.name)
		})
	}
}

func TestFeatureUpdateMBMFInstlFloors(t *testing.T) {
	type fields struct {
		cache            cache.Cache
		publisherFeature map[int]map[int]models.FeatureData
		mbmf             *mbmf
	}
	tests := []struct {
		name   string
		fields fields
		want   map[int]*models.MultiFloors // Expected floors in the active buffer
	}{
		{
			name: "empty publisherFeature map",
			fields: fields{
				cache:            nil,
				publisherFeature: map[int]map[int]models.FeatureData{},
				mbmf:             newMBMF(),
			},
			want: make(map[int]*models.MultiFloors),
		},
		{
			name: "instl floors enabled and disabled",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFInstlFloors: {
							Enabled: 1,
							Value:   `{"isActive":true,"tier1":1.5,"tier2":2.0,"tier3":2.5}`,
						},
					},
					456: {
						models.FeatureMBMFInstlFloors: {
							Enabled: 0,
						},
					},
				},
				mbmf: newMBMF(),
			},
			want: map[int]*models.MultiFloors{
				123: {
					IsActive: true,
					Tier1:    1.5,
					Tier2:    2.0,
					Tier3:    2.5,
				},
				456: {
					IsActive: false,
				},
			},
		},
		{
			name: "invalid json in floors value",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFInstlFloors: {
							Enabled: 1,
							Value:   `invalid json`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			want: make(map[int]*models.MultiFloors), // Should be empty due to invalid JSON
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				publisherFeature: tt.fields.publisherFeature,
				mbmf:             tt.fields.mbmf,
			}

			// Get current index and calculate next index
			currIdx := f.mbmf.index.Load()
			nextIdx := currIdx ^ 1

			// Update the next buffer
			f.updateMBMFInstlFloors(nextIdx)

			// Verify current buffer is untouched
			assert.Empty(t, f.mbmf.data[currIdx].instlFloors, "Current buffer should be untouched")

			// Switch to next buffer
			f.mbmf.index.Store(nextIdx)

			// Verify next buffer has expected floors
			assert.Equal(t, tt.want, f.mbmf.data[nextIdx].instlFloors, "Next buffer should have expected floors")
		})
	}
}

func TestFeatureUpdateMBMFRwddFloors(t *testing.T) {
	type fields struct {
		cache            cache.Cache
		publisherFeature map[int]map[int]models.FeatureData
		mbmf             *mbmf
	}
	tests := []struct {
		name   string
		fields fields
		want   map[int]*models.MultiFloors // Expected floors in the active buffer
	}{
		{
			name: "empty publisherFeature map",
			fields: fields{
				cache:            nil,
				publisherFeature: map[int]map[int]models.FeatureData{},
				mbmf:             newMBMF(),
			},
			want: make(map[int]*models.MultiFloors),
		},
		{
			name: "rwdd floors enabled and disabled",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFRwddFloors: {
							Enabled: 1,
							Value:   `{"isActive":true,"tier1":2.5,"tier2":3.0,"tier3":3.5}`,
						},
					},
					456: {
						models.FeatureMBMFRwddFloors: {
							Enabled: 0,
						},
					},
				},
				mbmf: newMBMF(),
			},
			want: map[int]*models.MultiFloors{
				123: {
					IsActive: true,
					Tier1:    2.5,
					Tier2:    3.0,
					Tier3:    3.5,
				},
				456: {
					IsActive: false,
				},
			},
		},
		{
			name: "invalid json in floors value",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFRwddFloors: {
							Enabled: 1,
							Value:   `invalid json`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			want: make(map[int]*models.MultiFloors), // Should be empty due to invalid JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				publisherFeature: tt.fields.publisherFeature,
				mbmf:             tt.fields.mbmf,
			}

			// Get current index and calculate next index
			currIdx := f.mbmf.index.Load()
			nextIdx := currIdx ^ 1

			// Update the next buffer
			f.updateMBMFRwddFloors(nextIdx)

			// Verify current buffer is untouched
			assert.Empty(t, f.mbmf.data[currIdx].rwddFloors, "Current buffer should be untouched")

			// Switch to next buffer
			f.mbmf.index.Store(nextIdx)

			// Verify next buffer has expected floors
			assert.Equal(t, tt.want, f.mbmf.data[nextIdx].rwddFloors, "Next buffer should have expected floors")
		})
	}
}

func TestFeatureUpdateMBMFBannerFloors(t *testing.T) {
	type fields struct {
		cache            cache.Cache
		publisherFeature map[int]map[int]models.FeatureData
		mbmf             *mbmf
	}
	tests := []struct {
		name   string
		fields fields
		want   map[int]*models.MultiFloors // Expected floors in the active buffer
	}{
		{
			name: "empty publisherFeature map",
			fields: fields{
				cache:            nil,
				publisherFeature: map[int]map[int]models.FeatureData{},
				mbmf:             newMBMF(),
			},
			want: make(map[int]*models.MultiFloors),
		},
		{
			name: "banner floors enabled and disabled",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFBannerFloors: {
							Enabled: 1,
							Value:   `{"isActive":true,"tier1":2.5,"tier2":3.0,"tier3":3.5}`,
						},
					},
					456: {
						models.FeatureMBMFBannerFloors: {
							Enabled: 0,
						},
					},
				},
				mbmf: newMBMF(),
			},
			want: map[int]*models.MultiFloors{
				123: {
					IsActive: true,
					Tier1:    2.5,
					Tier2:    3.0,
					Tier3:    3.5,
				},
				456: {
					IsActive: false,
				},
			},
		},
		{
			name: "invalid json in floors value",
			fields: fields{
				cache: nil,
				publisherFeature: map[int]map[int]models.FeatureData{
					123: {
						models.FeatureMBMFBannerFloors: {
							Enabled: 1,
							Value:   `invalid json`,
						},
					},
				},
				mbmf: newMBMF(),
			},
			want: make(map[int]*models.MultiFloors), // Should be empty due to invalid JSON
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				cache:            tt.fields.cache,
				publisherFeature: tt.fields.publisherFeature,
				mbmf:             tt.fields.mbmf,
			}
			// Get current index
			currIdx := f.mbmf.index.Load()
			// Update using next index (currIdx ^ 1)
			f.updateMBMFBannerFloors(currIdx ^ 1)
			// Store the next index
			f.mbmf.index.Store(currIdx ^ 1)
			// Assert using the new index
			assert.Equal(t, tt.want, f.mbmf.data[currIdx^1].bannerFloors, tt.name)
		})
	}
}

func TestFeatureIsMBMFCountryForPublisher(t *testing.T) {
	type fields struct {
		mbmf *mbmf
	}
	type args struct {
		countryCode string
		pubID       int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "mbmf country enabled pub with Country India",
			args: args{
				countryCode: "IN",
				pubID:       123,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledCountries: map[int]models.HashSet{
								123: {
									"IN": {},
									"US": {},
									"JP": {},
								},
								0: {
									"US": {},
								},
							},
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "country not present in pub specific country list",
			args: args{
				countryCode: "IN",
				pubID:       123,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledCountries: map[int]models.HashSet{
								123: {
									"US": {},
									"JP": {},
								},
								0: {
									"IN": {},
								},
							},
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "country present in pub specific country list",
			args: args{
				countryCode: "JP",
				pubID:       123,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledCountries: map[int]models.HashSet{
								123: {
									"IN": {},
									"US": {},
									"JP": {},
								},
								0: {
									"US": {},
								},
							},
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "country present in common country list for new pub",
			args: args{
				countryCode: "US",
				pubID:       125,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledCountries: map[int]models.HashSet{
								123: {
									"IN": {},
								},
								0: {
									"US": {},
								},
							},
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "mbmf disabled country pub request",
			args: args{
				countryCode: "DE",
				pubID:       125,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledCountries: map[int]models.HashSet{
								123: {
									"IN": {},
								},
								0: {
									"US": {},
								},
							},
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				mbmf: tt.fields.mbmf,
			}
			got := f.IsMBMFCountryForPublisher(tt.args.countryCode, tt.args.pubID)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestFeatureIsMBMFPublisherEnabled(t *testing.T) {
	type fields struct {
		mbmf *mbmf
	}
	type args struct {
		pubID int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "mbmf publisher enabled pub",
			args: args{
				pubID: 123,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledPublishers: map[int]bool{
								123: true,
							},
							enabledCountries:         make(map[int]models.HashSet),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "mbmf publisher disabled pub",
			args: args{
				pubID: 456,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledPublishers: map[int]bool{
								456: false,
							},
							enabledCountries:         make(map[int]models.HashSet),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "publisher not present in DB",
			args: args{
				pubID: 789,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							enabledPublishers: map[int]bool{
								123: true,
							},
							enabledCountries:         make(map[int]models.HashSet),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				mbmf: tt.fields.mbmf,
			}
			got := f.IsMBMFPublisherEnabled(tt.args.pubID)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestFeatureIsMBMFEnabledForAdUnitFormat(t *testing.T) {
	type fields struct {
		mbmf *mbmf
	}
	type args struct {
		pubID        int
		adunitFormat string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "empty adunitformat",
			args: args{
				pubID:        123,
				adunitFormat: "",
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: true,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "mbmf publisher enabled for instl",
			args: args{
				pubID:        123,
				adunitFormat: models.AdUnitFormatInstl,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: true,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "mbmf publisher enabled for rwdd",
			args: args{
				pubID:        1234,
				adunitFormat: models.AdUnitFormatRwddVideo,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							rwddFloors: map[int]*models.MultiFloors{
								1234: {
									IsActive: true,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "mbmf publisher disabled for instl",
			args: args{
				pubID:        456,
				adunitFormat: models.AdUnitFormatInstl,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								456: {
									IsActive: false,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "mbmf publisher not present in DB",
			args: args{
				pubID:        789,
				adunitFormat: models.AdUnitFormatInstl,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: true,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "mbmf publisher enabled for banner",
			args: args{
				pubID:        123,
				adunitFormat: models.AdUnitFormatBanner,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							bannerFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: true,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "mbmf publisher not present for banner",
			args: args{
				pubID:        123,
				adunitFormat: models.AdUnitFormatBanner,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							bannerFloors: map[int]*models.MultiFloors{},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "mbmf publisher disabled for banner",
			args: args{
				pubID:        123,
				adunitFormat: models.AdUnitFormatBanner,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							bannerFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: false,
								},
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				mbmf: tt.fields.mbmf,
			}
			got := f.IsMBMFEnabledForAdUnitFormat(tt.args.pubID, tt.args.adunitFormat)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestFeatureGetMBMFFloorsForAdUnitFormat(t *testing.T) {
	type fields struct {
		mbmf *mbmf
	}
	type args struct {
		pubID        int
		adunitFormat string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *models.MultiFloors
	}{
		{
			name: "empty adunitformat",
			args: args{
				pubID:        123,
				adunitFormat: "",
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: true,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "mbmf publisher for instl return as is",
			args: args{
				pubID:        123,
				adunitFormat: models.AdUnitFormatInstl,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: true,
									Tier1:    1,
									Tier2:    3,
									Tier3:    2,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1,
				Tier2:    3,
				Tier3:    2,
			},
		},
		{
			name: "mbmf publisher enabled for rwdd return as is",
			args: args{
				pubID:        1234,
				adunitFormat: models.AdUnitFormatRwddVideo,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							rwddFloors: map[int]*models.MultiFloors{
								1234: {
									IsActive: true,
									Tier1:    1,
									Tier2:    3,
									Tier3:    2,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1,
				Tier2:    3,
				Tier3:    2,
			},
		},
		{
			name: "mbmf publisher disabled for instl return as is",
			args: args{
				pubID:        456,
				adunitFormat: models.AdUnitFormatInstl,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								456: {
									IsActive: false,
									Tier1:    1,
									Tier2:    3,
									Tier3:    2,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: &models.MultiFloors{
				IsActive: false,
				Tier1:    1,
				Tier2:    3,
				Tier3:    2,
			},
		},
		{
			name: "publisher not present in DB,return default",
			args: args{
				pubID:        789,
				adunitFormat: models.AdUnitFormatInstl,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								models.DefaultAdUnitFormatFloors: {
									IsActive: true,
									Tier1:    1,
									Tier2:    2,
									Tier3:    3,
								},
								123: {
									IsActive: true,
									Tier1:    1,
									Tier2:    3,
									Tier3:    2,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1,
				Tier2:    2,
				Tier3:    3,
			},
		},
		{
			name: "publisher not present and default floors missing",
			args: args{
				pubID:        789,
				adunitFormat: models.AdUnitFormatInstl,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							instlFloors: map[int]*models.MultiFloors{
								123: {
									IsActive: true,
									Tier1:    1,
									Tier2:    3,
									Tier3:    2,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "mbmf publisher enabled for banner- don't apply floors",
			args: args{
				pubID:        1234,
				adunitFormat: models.AdUnitFormatBanner,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							bannerFloors: map[int]*models.MultiFloors{
								1234: {
									IsActive: true,
									Tier1:    1,
									Tier2:    3,
									Tier3:    2,
								},
								models.DefaultAdUnitFormatFloors: {
									IsActive: true,
									Tier1:    1,
									Tier2:    2,
									Tier3:    3,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "mbmf publisher not present for banner, don't apply any floors",
			args: args{
				pubID:        1234343,
				adunitFormat: models.AdUnitFormatBanner,
			},
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							bannerFloors: map[int]*models.MultiFloors{
								1234: {
									IsActive: true,
									Tier1:    1,
									Tier2:    3,
									Tier3:    2,
								},
							},
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
							bannerFloors:             make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				mbmf: tt.fields.mbmf,
			}
			got := f.GetMBMFFloorsForAdUnitFormat(tt.args.pubID, tt.args.adunitFormat)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestFeatureGetProfileAdUnitMultiFloors(t *testing.T) {
	type fields struct {
		mbmf *mbmf
	}
	type args struct {
		profileID int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]*models.MultiFloors
	}{
		{
			name: "profileID present",
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							profileAdUnitLevelFloors: models.ProfileAdUnitMultiFloors{
								123: {
									"adunit1": {
										IsActive: true,
										Tier1:    1,
										Tier2:    2,
										Tier3:    3,
									},
								},
							},
							enabledCountries:  make(map[int]models.HashSet),
							enabledPublishers: make(map[int]bool),
							instlFloors:       make(map[int]*models.MultiFloors),
							rwddFloors:        make(map[int]*models.MultiFloors),
						},
						{
							enabledCountries:         make(map[int]models.HashSet),
							enabledPublishers:        make(map[int]bool),
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
							instlFloors:              make(map[int]*models.MultiFloors),
							rwddFloors:               make(map[int]*models.MultiFloors),
						},
					},
				},
			},
			args: args{
				profileID: 123,
			},
			want: map[string]*models.MultiFloors{
				"adunit1": {
					IsActive: true,
					Tier1:    1,
					Tier2:    2,
					Tier3:    3,
				},
			},
		},
		{
			name: "profileID not present",
			fields: fields{
				mbmf: &mbmf{
					data: [2]mbmfData{
						{
							profileAdUnitLevelFloors: models.ProfileAdUnitMultiFloors{
								123: {
									"adunit1": {
										IsActive: true,
										Tier1:    1,
										Tier2:    2,
										Tier3:    3,
									},
								},
							},
						},
						{
							profileAdUnitLevelFloors: make(models.ProfileAdUnitMultiFloors),
						},
					},
				},
			},
			args: args{
				profileID: 456,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &feature{
				mbmf: tt.fields.mbmf,
			}
			got := f.GetProfileAdUnitMultiFloors(tt.args.profileID)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func TestFeatureMBMFConcurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mock_cache.NewMockCache(ctrl)

	// Setup feature with test data
	fe := &feature{
		cache: mockCache,
		publisherFeature: map[int]map[int]models.FeatureData{
			5890: {
				models.FeatureMBMFCountry: {
					Enabled: 1,
					Value:   `US`,
				},
			},
		},
		mbmf: newMBMF(),
	}

	// Initial index should be 0
	initialIndex := fe.mbmf.index.Load()
	assert.Equal(t, int32(0), initialIndex)

	// Setup mock expectations
	mockCache.EXPECT().GetProfileAdUnitMultiFloors().Return(models.ProfileAdUnitMultiFloors{}, nil).AnyTimes()

	// Run concurrent operations
	go func() {
		// Reader
		for i := 0; i < 100; i++ {
			fe.IsMBMFCountryForPublisher("US", 5890)
			// Verify index is either 0 or 1
			currIndex := fe.mbmf.index.Load()
			assert.True(t, currIndex == 0 || currIndex == 1)
		}
	}()

	// Writer
	for i := 0; i < 100; i++ {
		// Before update, index should be either 0 or 1
		prevIndex := fe.mbmf.index.Load()
		assert.True(t, prevIndex == 0 || prevIndex == 1)

		fe.updateMBMF()

		// After update, index should be flipped
		currIndex := fe.mbmf.index.Load()
		assert.Equal(t, prevIndex^1, currIndex)
	}
}

func TestExtractMultiFloors(t *testing.T) {
	type args struct {
		feature    map[int]models.FeatureData
		featureKey int
		pubID      int
	}
	tests := []struct {
		name string
		args args
		want *models.MultiFloors
	}{
		{
			name: "empty feature map",
			args: args{
				feature:    map[int]models.FeatureData{},
				featureKey: models.FeatureMBMFInstlFloors,
				pubID:      123,
			},
			want: nil,
		},
		{
			name: "feature name not present",
			args: args{
				feature: map[int]models.FeatureData{
					12345: {
						Value: "somevalue",
					},
				},
				featureKey: models.FeatureMBMFInstlFloors,
				pubID:      123,
			},
			want: nil,
		},
		{
			name: "value not present",
			args: args{
				feature: map[int]models.FeatureData{
					models.FeatureMBMFInstlFloors: {
						Enabled: 1,
					},
				},
				featureKey: models.FeatureMBMFInstlFloors,
				pubID:      123,
			},
			want: nil,
		},
		{
			name: "invalid json",
			args: args{
				feature: map[int]models.FeatureData{
					models.FeatureMBMFInstlFloors: {
						Enabled: 1,
						Value:   "invalidjson",
					},
				},
				featureKey: models.FeatureMBMFInstlFloors,
				pubID:      123,
			},
			want: nil,
		},
		{
			name: "valid jso for instl",
			args: args{
				feature: map[int]models.FeatureData{
					models.FeatureMBMFInstlFloors: {
						Enabled: 1,
						Value:   `{"isActive":true,"tier1":1.5,"tier2":2.0,"tier3":2.5}`,
					},
				},
				featureKey: models.FeatureMBMFInstlFloors,
				pubID:      123,
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1.5,
				Tier2:    2.0,
				Tier3:    2.5,
			},
		},
		{
			name: "valid json for rwdd",
			args: args{
				feature: map[int]models.FeatureData{
					models.FeatureMBMFRwddFloors: {
						Enabled: 1,
						Value:   `{"isActive":true,"tier1":1.5,"tier2":2.0,"tier3":2.5}`,
					},
				},
				featureKey: models.FeatureMBMFRwddFloors,
				pubID:      123,
			},
			want: &models.MultiFloors{
				IsActive: true,
				Tier1:    1.5,
				Tier2:    2.0,
				Tier3:    2.5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMultiFloors(tt.args.feature, tt.args.featureKey, tt.args.pubID)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
