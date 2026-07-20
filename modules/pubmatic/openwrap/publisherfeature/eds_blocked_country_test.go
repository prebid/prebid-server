package publisherfeature

import (
	"testing"

	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/stretchr/testify/assert"
)

func TestNewEDSBlockedCountries(t *testing.T) {
	got := newEDSBlockedCountries()

	assert.Equal(t, 0, got.index)
	assert.NotNil(t, got.countries[0])
	assert.NotNil(t, got.countries[1])
	assert.Empty(t, got.countries[0])
	assert.Empty(t, got.countries[1])
}

func TestUpdateEDSBlockedCountries(t *testing.T) {
	tests := []struct {
		name              string
		publisherFeature  map[int]map[int]models.FeatureData
		initialIndex      int
		initialCountries  [2]models.HashSet
		expectedIndex     int
		expectedCountries models.HashSet
		verifyInactive    models.HashSet
	}{
		{
			name:              "nil publisherFeature",
			publisherFeature:  nil,
			initialIndex:      0,
			initialCountries:  [2]models.HashSet{{"IN": {}}, {"US": {}}},
			expectedIndex:     0,
			expectedCountries: models.HashSet{"IN": {}},
			verifyInactive:    models.HashSet{"US": {}},
		},
		{
			name: "pubID=0 with enabled blocked countries",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   "IN,US,JP",
					},
				},
			},
			initialIndex:     0,
			initialCountries: [2]models.HashSet{{}, {}},
			expectedIndex:    1,
			expectedCountries: models.HashSet{
				"IN": {},
				"US": {},
				"JP": {},
			},
		},
		{
			name: "trims spaces and skips empty entries",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   " IN , US , , JP ",
					},
				},
			},
			initialIndex:     0,
			initialCountries: [2]models.HashSet{{}, {}},
			expectedIndex:    1,
			expectedCountries: models.HashSet{
				"IN": {},
				"US": {},
				"JP": {},
			},
		},
		{
			name: "enabled with empty value",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   "",
					},
				},
			},
			initialIndex:      0,
			initialCountries:  [2]models.HashSet{{}, {}},
			expectedIndex:     1,
			expectedCountries: models.HashSet{},
		},
		{
			name: "enabled with whitespace-only value",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   " , , ",
					},
				},
			},
			initialIndex:      0,
			initialCountries:  [2]models.HashSet{{}, {}},
			expectedIndex:     1,
			expectedCountries: models.HashSet{},
		},
		{
			name: "pubID=0 with disabled feature",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 0,
						Value:   "IN,US,JP",
					},
				},
			},
			initialIndex:      0,
			initialCountries:  [2]models.HashSet{{}, {}},
			expectedIndex:     1,
			expectedCountries: models.HashSet{},
		},
		{
			name: "publisher-specific row is ignored",
			publisherFeature: map[int]map[int]models.FeatureData{
				5890: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   "CN",
					},
				},
			},
			initialIndex:      0,
			initialCountries:  [2]models.HashSet{{}, {}},
			expectedIndex:     1,
			expectedCountries: models.HashSet{},
		},
		{
			name: "wildcard blocks all countries",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   models.EDSBlockedCountriesAll,
					},
				},
			},
			initialIndex:     0,
			initialCountries: [2]models.HashSet{{}, {}},
			expectedIndex:    1,
			expectedCountries: models.HashSet{
				models.EDSBlockedCountriesAll: {},
			},
		},
		{
			name: "toggle updates inactive buffer",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   "IN,US",
					},
				},
			},
			initialIndex: 1,
			initialCountries: [2]models.HashSet{
				{},
				{"DE": {}},
			},
			expectedIndex: 0,
			expectedCountries: models.HashSet{
				"IN": {},
				"US": {},
			},
			verifyInactive: models.HashSet{"DE": {}},
		},
		{
			name:              "no platform entry in DB",
			publisherFeature:  map[int]map[int]models.FeatureData{},
			initialIndex:      0,
			initialCountries:  [2]models.HashSet{{}, {}},
			expectedIndex:     1,
			expectedCountries: models.HashSet{},
		},
		{
			name: "pubID=0 present without EDS blocked countries feature",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {},
			},
			initialIndex:      0,
			initialCountries:  [2]models.HashSet{{}, {}},
			expectedIndex:     1,
			expectedCountries: models.HashSet{},
		},
		{
			name: "second update replaces previous countries",
			publisherFeature: map[int]map[int]models.FeatureData{
				0: {
					models.FeatureEDSBlockedCountries: {
						Enabled: 1,
						Value:   "FR",
					},
				},
			},
			initialIndex: 1,
			initialCountries: [2]models.HashSet{
				{"IN": {}, "US": {}},
				{},
			},
			expectedIndex: 0,
			expectedCountries: models.HashSet{
				"FR": {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := &feature{
				publisherFeature: tt.publisherFeature,
				edsBlockedCountries: edsBlockedCountries{
					countries: tt.initialCountries,
					index:     tt.initialIndex,
				},
			}
			fe.updateEDSBlockedCountries()
			assert.Equal(t, tt.expectedIndex, fe.edsBlockedCountries.index)
			assert.Equal(t, tt.expectedCountries, fe.edsBlockedCountries.countries[tt.expectedIndex])
			if tt.verifyInactive != nil {
				inactiveIndex := tt.expectedIndex ^ 1
				assert.Equal(t, tt.verifyInactive, fe.edsBlockedCountries.countries[inactiveIndex])
			}
		})
	}
}

func TestUpdateEDSBlockedCountries_doubleUpdate(t *testing.T) {
	fe := &feature{
		publisherFeature: map[int]map[int]models.FeatureData{
			0: {
				models.FeatureEDSBlockedCountries: {
					Enabled: 1,
					Value:   "IN,US",
				},
			},
		},
		edsBlockedCountries: newEDSBlockedCountries(),
	}

	fe.updateEDSBlockedCountries()
	assert.True(t, fe.IsEDSBlockedCountry("IN"))
	assert.True(t, fe.IsEDSBlockedCountry("US"))
	assert.False(t, fe.IsEDSBlockedCountry("DE"))

	fe.publisherFeature[0][models.FeatureEDSBlockedCountries] = models.FeatureData{
		Enabled: 1,
		Value:   "DE",
	}
	fe.updateEDSBlockedCountries()
	assert.False(t, fe.IsEDSBlockedCountry("IN"))
	assert.False(t, fe.IsEDSBlockedCountry("US"))
	assert.True(t, fe.IsEDSBlockedCountry("DE"))
}

func TestIsEDSBlockedCountry(t *testing.T) {
	tests := []struct {
		name        string
		countryCode string
		index       int
		countries   [2]models.HashSet
		want        bool
	}{
		{
			name:        "blocked country",
			countryCode: "IN",
			index:       0,
			countries: [2]models.HashSet{
				{"IN": {}, "US": {}, "JP": {}},
				{},
			},
			want: true,
		},
		{
			name:        "non-blocked country",
			countryCode: "DE",
			index:       0,
			countries: [2]models.HashSet{
				{"IN": {}, "US": {}, "JP": {}},
				{},
			},
			want: false,
		},
		{
			name:        "reads active buffer when index is 1",
			countryCode: "US",
			index:       1,
			countries: [2]models.HashSet{
				{"IN": {}},
				{"US": {}, "JP": {}},
			},
			want: true,
		},
		{
			name:        "inactive buffer is ignored when index is 1",
			countryCode: "IN",
			index:       1,
			countries: [2]models.HashSet{
				{"IN": {}},
				{"US": {}},
			},
			want: false,
		},
		{
			name:        "wildcard blocks any country",
			countryCode: "DE",
			index:       0,
			countries: [2]models.HashSet{
				{models.EDSBlockedCountriesAll: {}},
				{},
			},
			want: true,
		},
		{
			name:        "wildcard blocks empty country code",
			countryCode: "",
			index:       0,
			countries: [2]models.HashSet{
				{models.EDSBlockedCountriesAll: {}},
				{},
			},
			want: true,
		},
		{
			name:        "empty country code is not blocked",
			countryCode: "",
			index:       0,
			countries: [2]models.HashSet{
				{"IN": {}},
				{},
			},
			want: false,
		},
		{
			name:        "empty country list does not block any country",
			countryCode: "IN",
			index:       0,
			countries: [2]models.HashSet{
				{},
				{},
			},
			want: false,
		},
		{
			name:        "empty country list does not block empty country code",
			countryCode: "",
			index:       0,
			countries: [2]models.HashSet{
				{},
				{},
			},
			want: false,
		},
		{
			name:        "country code match is case-sensitive",
			countryCode: "in",
			index:       0,
			countries: [2]models.HashSet{
				{"IN": {}},
				{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := &feature{
				edsBlockedCountries: edsBlockedCountries{
					countries: tt.countries,
					index:     tt.index,
				},
			}
			got := fe.IsEDSBlockedCountry(tt.countryCode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsEDSBlockedCountry_noDBEntry(t *testing.T) {
	fe := &feature{
		publisherFeature:    map[int]map[int]models.FeatureData{},
		edsBlockedCountries: newEDSBlockedCountries(),
	}
	fe.updateEDSBlockedCountries()

	assert.False(t, fe.IsEDSBlockedCountry("IN"))
	assert.False(t, fe.IsEDSBlockedCountry("US"))
	assert.False(t, fe.IsEDSBlockedCountry(""))
}
