package openwrap

import (
	"testing"

	"github.com/buger/jsonparser"
	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache"
	mock_cache "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrichApsRequest(t *testing.T) {
	uuid1 := "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	uuidA := "d9428888-122b-41e1-b85f-6f5d6f0f8b7a"
	uuidB := "1b4e28ba-2fa1-41d2-883f-0016d3cca427"
	unmapped := "7f8a9b0c-11d2-43e4-9a5b-6c7d8e9f0a1b"

	tests := []struct {
		name        string
		body        []byte
		publisherID string
		cacheNil    bool
		setupMock   func(m *mock_cache.MockCache)
		wantErr     bool
		wantNBR     openrtb3.NoBidReason
		checkOut    func(t *testing.T, out []byte)
	}{
		{
			name:        "success_single_imp",
			body:        []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"` + uuid1 + `","banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"1"}}}`),
			publisherID: "1",
			setupMock: func(m *mock_cache.MockCache) {
				m.EXPECT().GetApsOwMapping(uuid1).Return("ow-ad-unit-name-1", 10042, true)
			},
			wantNBR: 0,
			checkOut: func(t *testing.T, out []byte) {
				assert.Equal(t, "ow-ad-unit-name-1", apsTestJSONString(t, out, "imp", "[0]", "tagid"))
				assert.Equal(t, uuid1, apsTestJSONString(t, out, "imp", "[0]", "ext", "gpid"))
				assert.Equal(t, int64(10042), apsTestJSONInt64(t, out, "ext", "prebid", "bidderparams", "pubmatic", "wrapper", "profileid"))
			},
		},
		{
			name:        "success_preserves_existing_gpid",
			body:        []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"` + uuid1 + `","ext":{"gpid":"existing-gpid"},"banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"1"}}}`),
			publisherID: "1",
			setupMock: func(m *mock_cache.MockCache) {
				m.EXPECT().GetApsOwMapping(uuid1).Return("ow-ad-unit-name-1", 10042, true)
			},
			wantNBR: 0,
			checkOut: func(t *testing.T, out []byte) {
				assert.Equal(t, "existing-gpid", apsTestJSONString(t, out, "imp", "[0]", "ext", "gpid"))
			},
		},
		{
			name:        "success_overwrites_empty_gpid",
			body:        []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"` + uuid1 + `","ext":{"gpid":""},"banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"1"}}}`),
			publisherID: "1",
			setupMock: func(m *mock_cache.MockCache) {
				m.EXPECT().GetApsOwMapping(uuid1).Return("ow-ad-unit-name-1", 10042, true)
			},
			wantNBR: 0,
			checkOut: func(t *testing.T, out []byte) {
				assert.Equal(t, uuid1, apsTestJSONString(t, out, "imp", "[0]", "ext", "gpid"))
			},
		},
		{
			name:        "success_multi_imp_only_first_imp_enriched",
			body:        []byte(`{"id":"r2","imp":[{"id":"1","tagid":"` + uuidA + `"},{"id":"2","tagid":"` + uuidB + `"}],"app":{"publisher":{"id":"1"}}}`),
			publisherID: "1",
			setupMock: func(m *mock_cache.MockCache) {
				m.EXPECT().GetApsOwMapping(uuidA).Return("ow-a-name", 10042, true)
			},
			wantNBR: 0,
			checkOut: func(t *testing.T, out []byte) {
				assert.Equal(t, "ow-a-name", apsTestJSONString(t, out, "imp", "[0]", "tagid"))
				assert.Equal(t, uuidB, apsTestJSONString(t, out, "imp", "[1]", "tagid"))
				assert.Equal(t, int64(10042), apsTestJSONInt64(t, out, "ext", "prebid", "bidderparams", "pubmatic", "wrapper", "profileid"))
			},
		},
		{
			name:        "err_unmapped_uuid",
			body:        []byte(`{"imp":[{"id":"1","tagid":"` + unmapped + `"}],"app":{"publisher":{"id":"1"}}}`),
			publisherID: "1",
			setupMock:   func(m *mock_cache.MockCache) { m.EXPECT().GetApsOwMapping(unmapped).Return("", 0, false) },
			wantErr:     true,
			wantNBR:     nbr.APSSlotUUIDNotMapped,
		},
		{
			name:        "err_non_uuid_tagid_unmapped",
			body:        []byte(`{"imp":[{"id":"1","tagid":"unknown-uuid"}],"app":{"publisher":{"id":"1"}}}`),
			publisherID: "1",
			setupMock:   func(m *mock_cache.MockCache) { m.EXPECT().GetApsOwMapping("unknown-uuid").Return("", 0, false) },
			wantErr:     true,
			wantNBR:     nbr.APSSlotUUIDNotMapped,
		},
		{
			name:        "err_nil_cache",
			body:        []byte(`{"imp":[{"tagid":"` + uuid1 + `"}]}`),
			publisherID: "0",
			cacheNil:    true,
			wantErr:     true,
			wantNBR:     nbr.InternalError,
		},
		{
			name:        "err_no_impressions",
			body:        []byte(`{"imp":[]}`),
			publisherID: "1",
			wantErr:     true,
			wantNBR:     openrtb3.NoBidInvalidRequest,
		},
		{
			name:        "err_empty_tagid",
			body:        []byte(`{"imp":[{"id":"1","tagid":""}]}`),
			publisherID: "1",
			wantErr:     true,
			wantNBR:     nbr.InvalidImpressionTagID,
		},
		{
			name:        "err_invalid_json",
			body:        []byte(`not json`),
			publisherID: "1",
			wantErr:     true,
			wantNBR:     openrtb3.NoBidInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			var owCache cache.Cache
			if tt.cacheNil {
				owCache = nil
			} else {
				m := mock_cache.NewMockCache(ctrl)
				if tt.setupMock != nil {
					tt.setupMock(m)
				}
				owCache = m
			}

			out, err, gotNBR := enrichApsRequest(tt.body, owCache, nil, tt.publisherID)

			assert.Equal(t, tt.wantNBR, gotNBR, "NoBidReason (3rd return)")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.checkOut != nil {
				tt.checkOut(t, out)
			}
		})
	}
}

func apsTestJSONString(t *testing.T, out []byte, keys ...string) string {
	t.Helper()
	s, err := jsonparser.GetString(out, keys...)
	require.NoError(t, err)
	return s
}

func apsTestJSONInt64(t *testing.T, out []byte, keys ...string) int64 {
	t.Helper()
	v, err := jsonparser.GetInt(out, keys...)
	require.NoError(t, err)
	return v
}
