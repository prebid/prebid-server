package privacy

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestScrubDeviceIDs(t *testing.T) {
	testCases := []struct {
		name           string
		deviceIn       *openrtb2.Device
		expectedDevice *openrtb2.Device
	}{
		{
			name:           "all",
			deviceIn:       &openrtb2.Device{DIDMD5: "MD5", DIDSHA1: "SHA1", DPIDMD5: "MD5", DPIDSHA1: "SHA1", IFA: "IFA", MACMD5: "MD5", MACSHA1: "SHA1"},
			expectedDevice: &openrtb2.Device{DIDMD5: "", DIDSHA1: "", DPIDMD5: "", DPIDSHA1: "", IFA: "", MACMD5: "", MACSHA1: ""},
		},
		{
			name:           "nil",
			deviceIn:       nil,
			expectedDevice: nil,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Device: test.deviceIn}}
			scrubDeviceIDs(brw)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedDevice, brw.Device)
		})
	}
}

func TestScrubUserIDs(t *testing.T) {
	testCases := []struct {
		name         string
		userIn       *openrtb2.User
		expectedUser *openrtb2.User
	}{
		{
			name:         "all",
			userIn:       &openrtb2.User{Data: []openrtb2.Data{}, ID: "ID", BuyerUID: "bID", Yob: 2000, Gender: "M", Keywords: "keywords", KwArray: nil},
			expectedUser: &openrtb2.User{Data: nil, ID: "", BuyerUID: "", Yob: 0, Gender: "", Keywords: "", KwArray: nil},
		},
		{
			name:         "nil",
			userIn:       nil,
			expectedUser: nil,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{User: test.userIn}}
			scrubUserIDs(brw)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedUser, brw.User)
		})
	}
}

func TestScrubUserDemographics(t *testing.T) {
	testCases := []struct {
		name         string
		userIn       *openrtb2.User
		expectedUser *openrtb2.User
	}{
		{
			name:         "all",
			userIn:       &openrtb2.User{ID: "ID", BuyerUID: "bID", Yob: 2000, Gender: "M"},
			expectedUser: &openrtb2.User{ID: "", BuyerUID: "", Yob: 0, Gender: ""},
		},
		{
			name:         "nil",
			userIn:       nil,
			expectedUser: nil,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{User: test.userIn}}
			scrubUserDemographics(brw)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedUser, brw.User)
		})
	}
}

func TestScrubUserExt(t *testing.T) {
	testCases := []struct {
		name         string
		userIn       *openrtb2.User
		fieldName    string
		expectedUser *openrtb2.User
	}{
		{
			name:         "nil_user",
			userIn:       nil,
			expectedUser: nil,
		},
		{
			name:         "nil_ext",
			userIn:       &openrtb2.User{ID: "ID", Ext: nil},
			expectedUser: &openrtb2.User{ID: "ID", Ext: nil},
		},
		{
			name:         "empty_ext",
			userIn:       &openrtb2.User{ID: "ID", Ext: json.RawMessage(`{}`)},
			expectedUser: &openrtb2.User{ID: "ID", Ext: json.RawMessage(`{}`)},
		},
		{
			name:         "ext_with_field",
			userIn:       &openrtb2.User{ID: "ID", Ext: json.RawMessage(`{"data":"123","test":1}`)},
			fieldName:    "data",
			expectedUser: &openrtb2.User{ID: "ID", Ext: json.RawMessage(`{"test":1}`)},
		},
		{
			name:         "ext_without_field",
			userIn:       &openrtb2.User{ID: "ID", Ext: json.RawMessage(`{"data":"123","test":1}`)},
			fieldName:    "noData",
			expectedUser: &openrtb2.User{ID: "ID", Ext: json.RawMessage(`{"data":"123","test":1}`)},
		},
		{
			name:         "nil",
			userIn:       nil,
			expectedUser: nil,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{User: test.userIn}}
			scrubUserExt(brw, test.fieldName)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedUser, brw.User)
		})
	}
}

func TestScrubEids(t *testing.T) {
	testCases := []struct {
		name         string
		userIn       *openrtb2.User
		expectedUser *openrtb2.User
	}{
		{
			name:         "eids",
			userIn:       &openrtb2.User{ID: "ID", EIDs: []openrtb2.EID{}},
			expectedUser: &openrtb2.User{ID: "ID", EIDs: nil},
		},
		{
			name:         "nil_eids",
			userIn:       &openrtb2.User{ID: "ID", EIDs: nil},
			expectedUser: &openrtb2.User{ID: "ID", EIDs: nil},
		},
		{
			name:         "nil",
			userIn:       nil,
			expectedUser: nil,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{User: test.userIn}}
			ScrubEIDs(brw)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedUser, brw.User)
		})
	}
}

func TestScrubTID(t *testing.T) {
	testCases := []struct {
		name           string
		sourceIn       *openrtb2.Source
		impIn          []openrtb2.Imp
		expectedSource *openrtb2.Source
		expectedImp    []openrtb2.Imp
	}{
		{
			name:           "nil",
			sourceIn:       nil,
			expectedSource: nil,
		},
		{
			name:           "nil_imp_ext",
			sourceIn:       &openrtb2.Source{TID: "tid"},
			impIn:          []openrtb2.Imp{{ID: "impID", Ext: nil}},
			expectedSource: &openrtb2.Source{TID: ""},
			expectedImp:    []openrtb2.Imp{{ID: "impID", Ext: nil}},
		},
		{
			name:           "empty_imp_ext",
			sourceIn:       &openrtb2.Source{TID: "tid"},
			impIn:          []openrtb2.Imp{{ID: "impID", Ext: json.RawMessage(`{}`)}},
			expectedSource: &openrtb2.Source{TID: ""},
			expectedImp:    []openrtb2.Imp{{ID: "impID", Ext: json.RawMessage(`{}`)}},
		},
		{
			name:           "ext_with_tid",
			sourceIn:       &openrtb2.Source{TID: "tid"},
			impIn:          []openrtb2.Imp{{ID: "impID", Ext: json.RawMessage(`{"tid":"123","test":1}`)}},
			expectedSource: &openrtb2.Source{TID: ""},
			expectedImp:    []openrtb2.Imp{{ID: "impID", Ext: json.RawMessage(`{"test":1}`)}},
		},
		{
			name:           "ext_without_tid",
			sourceIn:       &openrtb2.Source{TID: "tid"},
			impIn:          []openrtb2.Imp{{ID: "impID", Ext: json.RawMessage(`{"data":"123","test":1}`)}},
			expectedSource: &openrtb2.Source{TID: ""},
			expectedImp:    []openrtb2.Imp{{ID: "impID", Ext: json.RawMessage(`{"data":"123","test":1}`)}},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Source: test.sourceIn, Imp: test.impIn}}
			ScrubTID(brw)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedSource, brw.Source)
			assert.Equal(t, test.expectedImp, brw.Imp)
		})
	}
}

func TestScrubGEO(t *testing.T) {
	testCases := []struct {
		name           string
		userIn         *openrtb2.User
		expectedUser   *openrtb2.User
		deviceIn       *openrtb2.Device
		expectedDevice *openrtb2.Device
	}{
		{
			name:           "nil",
			userIn:         nil,
			expectedUser:   nil,
			deviceIn:       nil,
			expectedDevice: nil,
		},
		{
			name:           "nil_user_geo",
			userIn:         &openrtb2.User{ID: "ID", Geo: nil},
			expectedUser:   &openrtb2.User{ID: "ID", Geo: nil},
			deviceIn:       &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.12)}},
		},
		{
			name:           "with_user_geo",
			userIn:         &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedUser:   &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.12)}},
			deviceIn:       &openrtb2.Device{},
			expectedDevice: &openrtb2.Device{},
		},
		{
			name:           "nil_device_geo",
			userIn:         &openrtb2.User{},
			expectedUser:   &openrtb2.User{},
			deviceIn:       &openrtb2.Device{Geo: nil},
			expectedDevice: &openrtb2.Device{Geo: nil},
		},
		{
			name:           "with_device_geo",
			userIn:         &openrtb2.User{},
			expectedUser:   &openrtb2.User{},
			deviceIn:       &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.12)}},
		},
		{
			name:           "with_user_and_device_geo",
			userIn:         &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedUser:   &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.12)}},
			deviceIn:       &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.12)}},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{User: test.userIn, Device: test.deviceIn}}
			scrubGEO(brw)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedUser, brw.User)
			assert.Equal(t, test.expectedDevice, brw.Device)
		})
	}
}

func TestScrubGeoFull(t *testing.T) {
	testCases := []struct {
		name           string
		userIn         *openrtb2.User
		expectedUser   *openrtb2.User
		deviceIn       *openrtb2.Device
		expectedDevice *openrtb2.Device
	}{
		{
			name:           "nil",
			userIn:         nil,
			expectedUser:   nil,
			deviceIn:       nil,
			expectedDevice: nil,
		},
		{
			name:           "nil_user_geo",
			userIn:         &openrtb2.User{ID: "ID", Geo: nil},
			expectedUser:   &openrtb2.User{ID: "ID", Geo: nil},
			deviceIn:       &openrtb2.Device{},
			expectedDevice: &openrtb2.Device{},
		},
		{
			name:           "with_user_geo",
			userIn:         &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedUser:   &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{}},
			deviceIn:       &openrtb2.Device{},
			expectedDevice: &openrtb2.Device{},
		},
		{
			name:           "nil_device_geo",
			userIn:         &openrtb2.User{},
			expectedUser:   &openrtb2.User{},
			deviceIn:       &openrtb2.Device{Geo: nil},
			expectedDevice: &openrtb2.Device{Geo: nil},
		},
		{
			name:           "with_device_geo",
			userIn:         &openrtb2.User{},
			expectedUser:   &openrtb2.User{},
			deviceIn:       &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{}},
		},
		{
			name:           "with_user_and_device_geo",
			userIn:         &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedUser:   &openrtb2.User{ID: "ID", Geo: &openrtb2.Geo{}},
			deviceIn:       &openrtb2.Device{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(123.123)}},
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{}},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			brw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{User: test.userIn, Device: test.deviceIn}}
			scrubGeoFull(brw)
			brw.RebuildRequest()
			assert.Equal(t, test.expectedUser, brw.User)
			assert.Equal(t, test.expectedDevice, brw.Device)
		})
	}
}

func TestScrubIP(t *testing.T) {
	testCases := []struct {
		IP        string
		cleanedIP string
		bits      int
		maskBits  int
	}{
		{
			IP:        "0:0:0:0:0:0:0:0",
			cleanedIP: "::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "",
			cleanedIP: "",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "1111:2222:3333:4444:5555:6666:7777:8888",
			cleanedIP: "1111:2222:3333:4400::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "1111:2222:3333:4444:5555:6666:7777:8888",
			cleanedIP: "1111:2222::",
			bits:      128,
			maskBits:  34,
		},
		{
			IP:        "1111:0:3333:4444:5555:6666:7777:8888",
			cleanedIP: "1111:0:3333:4400::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "1111::6666:7777:8888",
			cleanedIP: "1111::",
			bits:      128,
			maskBits:  56,
		},
		{
			IP:        "2001:1db8:0000:0000:0000:ff00:0042:8329",
			cleanedIP: "2001:1db8::ff00:0:0",
			bits:      128,
			maskBits:  96,
		},
		{
			IP:        "2001:1db8:0000:0000:0000:ff00:0:0",
			cleanedIP: "2001:1db8::ff00:0:0",
			bits:      128,
			maskBits:  96,
		},
	}
	for _, test := range testCases {
		t.Run(test.IP, func(t *testing.T) {
			// bits: ipv6 - 128, ipv4 - 32
			result := scrubIP(test.IP, test.maskBits, test.bits)
			assert.Equal(t, test.cleanedIP, result)
		})
	}
}

func TestScrubGeoPrecision(t *testing.T) {
	geo := &openrtb2.Geo{
		Lat:   ptrutil.ToPtr(123.456),
		Lon:   ptrutil.ToPtr(678.89),
		Metro: "some metro",
		City:  "some city",
		ZIP:   "some zip",
	}
	geoExpected := &openrtb2.Geo{
		Lat:   ptrutil.ToPtr(123.46),
		Lon:   ptrutil.ToPtr(678.89),
		Metro: "some metro",
		City:  "some city",
		ZIP:   "some zip",
	}

	result := scrubGeoPrecision(geo)

	assert.Equal(t, geoExpected, result)
}

func TestScrubGeoPrecisionWhenNil(t *testing.T) {
	result := scrubGeoPrecision(nil)
	assert.Nil(t, result)
}

func TestScrubUserExtIDs(t *testing.T) {
	testCases := []struct {
		description string
		userExt     json.RawMessage
		expected    json.RawMessage
	}{
		{
			description: "Nil",
			userExt:     nil,
			expected:    nil,
		},
		{
			description: "Empty String",
			userExt:     json.RawMessage(``),
			expected:    json.RawMessage(``),
		},
		{
			description: "Empty Object",
			userExt:     json.RawMessage(`{}`),
			expected:    json.RawMessage(`{}`),
		},
		{
			description: "Do Nothing When Malformed",
			userExt:     json.RawMessage(`malformed`),
			expected:    json.RawMessage(`malformed`),
		},
		{
			description: "Do Nothing When No IDs Present",
			userExt:     json.RawMessage(`{"anyExisting":42}}`),
			expected:    json.RawMessage(`{"anyExisting":42}}`),
		},
		{
			description: "Remove eids",
			userExt:     json.RawMessage(`{"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{}`),
		},
		{
			description: "Remove eids - With Other Data",
			userExt:     json.RawMessage(`{"anyExisting":42,"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{"anyExisting":42}`),
		},
		{
			description: "Remove eids - With Other Nested Data",
			userExt:     json.RawMessage(`{"anyExisting":{"existing":42},"eids":[{"source":"anySource","id":"anyId","uids":[{"id":"anyId","ext":{"id":42}}],"ext":{"id":42}}]}`),
			expected:    json.RawMessage(`{"anyExisting":{"existing":42}}`),
		},
		{
			description: "Remove eids Only - Empty Array",
			userExt:     json.RawMessage(`{"eids":[]}`),
			expected:    json.RawMessage(`{}`),
		},
	}

	for _, test := range testCases {
		result := scrubExtIDs(test.userExt, "eids")
		assert.Equal(t, test.expected, result, test.description)
	}
}
