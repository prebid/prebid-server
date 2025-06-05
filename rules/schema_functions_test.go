package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/prebid/prebid-server/v3/util/randomutil"
	"github.com/stretchr/testify/assert"
)

func TestGetDeviceGeo(t *testing.T) {
	testCases := []struct {
		desc        string
		inWrapper   *openrtb_ext.RequestWrapper
		expectedGeo *openrtb2.Geo
	}{
		{
			desc:        "nil wrapper",
			inWrapper:   nil,
			expectedGeo: nil,
		},
		{
			desc:        "nil wrapper.bidRequest",
			inWrapper:   &openrtb_ext.RequestWrapper{},
			expectedGeo: nil,
		},
		{
			desc: "nil wrapper.bidRequest.device",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectedGeo: nil,
		},
		{
			desc: "nil wrapper.bidRequest.device.geo",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
			expectedGeo: nil,
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Country: "MEX",
						},
					},
				},
			},
			expectedGeo: &openrtb2.Geo{Country: "MEX"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			geo := getDeviceGeo(tc.inWrapper)
			assert.Equal(t, tc.expectedGeo, geo)
		})
	}
}

func TestGetExtRequestPrebid(t *testing.T) {
	testCases := []struct {
		desc           string
		inWrapper      *openrtb_ext.RequestWrapper
		expectedPrebid *openrtb_ext.ExtRequestPrebid
		expectedError  error
	}{
		{
			desc: "nil request.ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectedPrebid: nil,
			expectedError:  nil,
		},
		{
			desc: "malformed request.ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage("malformed"),
				},
			},
			expectedPrebid: nil,
			expectedError:  &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc: "empty request.ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage{},
				},
			},
			expectedPrebid: nil,
			expectedError:  nil,
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{}}`),
				},
			},
			expectedPrebid: &openrtb_ext.ExtRequestPrebid{},
			expectedError:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			prebid, err := getExtRequestPrebid(tc.inWrapper)
			assert.Equal(t, tc.expectedPrebid, prebid)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestGetUserEIDS(t *testing.T) {
	testCases := []struct {
		desc            string
		inWrapper       *openrtb_ext.RequestWrapper
		expectedEIDsArr []openrtb2.EID
	}{
		{
			desc:            "nil wrapper",
			inWrapper:       nil,
			expectedEIDsArr: nil,
		},
		{
			desc: "nil request.User",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectedEIDsArr: nil,
		},
		{
			desc: "nil request.User.EIDs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			expectedEIDsArr: nil,
		},
		{
			desc: "empty request.User.EIDs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{},
					},
				},
			},
			expectedEIDsArr: nil,
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "anySource",
							},
						},
					},
				},
			},
			expectedEIDsArr: []openrtb2.EID{
				{
					Source: "anySource",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			eids := getUserEIDS(tc.inWrapper)
			assert.Equal(t, tc.expectedEIDsArr, eids)
		})
	}
}

func TestNewDeviceCountryIn(t *testing.T) {
	testCases := []struct {
		desc               string
		inParams           json.RawMessage
		outDeviceCountryIn SchemaFunction[openrtb_ext.RequestWrapper]
		outErr             error
	}{
		{
			desc:               "nil json",
			inParams:           nil,
			outDeviceCountryIn: nil,
			outErr:             &errortypes.FailedToUnmarshal{Message: "expect { or n, but found \x00"},
		},
		{
			desc:               "malformed json",
			inParams:           json.RawMessage(`malformed`),
			outDeviceCountryIn: nil,
			outErr:             &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc:               "empty args array",
			inParams:           json.RawMessage(`{"countries":[]}`),
			outDeviceCountryIn: nil,
			outErr:             errors.New("Missing countries argument for deviceCountryIn schema function"),
		},
		{
			desc:     "array args",
			inParams: json.RawMessage(`{"countries":["JPN"]}`),
			outDeviceCountryIn: &deviceCountryIn{
				Countries: []string{"JPN"},
				CountryDirectory: map[string]struct{}{
					"JPN": {},
				},
			},
			outErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			deviceCountryIn, err := NewDeviceCountryIn(tc.inParams)

			assert.Equal(t, tc.outDeviceCountryIn, deviceCountryIn)
			assert.Equal(t, tc.outErr, err)
		})
	}
}

func TestDeviceCountryInCall(t *testing.T) {
	wrapperWithCountryCode := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Device: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Country: "MEX",
				},
			},
		},
	}

	testCases := []struct {
		desc               string
		inDeviceCountryIn  deviceCountryIn
		inRequestWrapper   *openrtb_ext.RequestWrapper
		expectedStringBool string
	}{
		{
			desc:               "nil wrapper.device.geo",
			inDeviceCountryIn:  deviceCountryIn{},
			inRequestWrapper:   nil,
			expectedStringBool: "false",
		},
		{
			desc:              "empty wrapper.device.geo.country",
			inDeviceCountryIn: deviceCountryIn{},
			inRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Country: "",
						},
					},
				},
			},
			expectedStringBool: "false",
		},
		{
			desc:               "empty country list",
			inDeviceCountryIn:  deviceCountryIn{},
			inRequestWrapper:   wrapperWithCountryCode,
			expectedStringBool: "false",
		},
		{
			desc: "wrapper.device.geo.country not found in country list",
			inDeviceCountryIn: deviceCountryIn{
				CountryDirectory: map[string]struct{}{
					"USA": {},
					"CAN": {},
				},
			},
			inRequestWrapper:   wrapperWithCountryCode,
			expectedStringBool: "false",
		},
		{
			desc: "success",
			inDeviceCountryIn: deviceCountryIn{
				CountryDirectory: map[string]struct{}{
					"USA": {},
					"MEX": {},
					"CAN": {},
				},
			},
			inRequestWrapper:   wrapperWithCountryCode,
			expectedStringBool: "true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := tc.inDeviceCountryIn.Call(tc.inRequestWrapper)
			assert.Equal(t, tc.expectedStringBool, result)
			assert.Nil(t, err)
		})
	}
}

func TestDeviceCountryCall(t *testing.T) {
	testCases := []struct {
		desc            string
		inWrapper       *openrtb_ext.RequestWrapper
		expectedCountry string
	}{
		{
			desc: "nil wrapper.bidRequest.device.geo",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
		},
		{
			desc: "empty wrapper.bidRequest.device.geo.country",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{},
					},
				},
			},
		},
		{
			desc: "valid wrapper.bidRequest.device.geo.country",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Country: "MEX",
						},
					},
				},
			},
			expectedCountry: "MEX",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dc := &deviceCountry{}

			country, err := dc.Call(tc.inWrapper)
			assert.Equal(t, tc.expectedCountry, country)
			assert.Nil(t, err)
		})
	}
}

func TestDataCenterCall(t *testing.T) {
	testCases := []struct {
		desc           string
		inWrapper      *openrtb_ext.RequestWrapper
		expectedRegion string
	}{
		{
			desc: "nil wrapper.bidRequest.device.geo",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
		},
		{
			desc: "empty wrapper.bidRequest.device.geo.region",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{},
					},
				},
			},
		},
		{
			desc: "valid wrapper.bidRequest.device.geo.region",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Region: "NorthAmerica",
						},
					},
				},
			},
			expectedRegion: "NorthAmerica",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dc := &dataCenter{}

			region, err := dc.Call(tc.inWrapper)
			assert.Equal(t, tc.expectedRegion, region)
			assert.Nil(t, err)
		})
	}
}

func TestDataCenterInCall(t *testing.T) {
	testCases := []struct {
		desc             string
		inDataCenterIn   dataCenterIn
		inRequestWrapper *openrtb_ext.RequestWrapper
		expectedResult   string
	}{
		{
			desc: "nil wrapper.device.geo",
			inRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
			expectedResult: "false",
		},
		{
			desc:           "empty wrapper.device.geo.region",
			inDataCenterIn: dataCenterIn{},
			inRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Region: "",
						},
					},
				},
			},
			expectedResult: "false",
		},
		{
			desc:           "empty region dir",
			inDataCenterIn: dataCenterIn{},
			inRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Region: "NorthAmerica",
						},
					},
				},
			},
			expectedResult: "false",
		},
		{
			desc: "wrapper.device.geo.region not in dir",
			inDataCenterIn: dataCenterIn{
				DataCenterDir: map[string]struct{}{
					"Europe": {},
					"Africa": {},
				},
			},
			inRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Region: "NorthAmerica",
						},
					},
				},
			},
			expectedResult: "false",
		},
		{
			desc: "success",
			inDataCenterIn: dataCenterIn{
				DataCenterDir: map[string]struct{}{
					"Europe":       {},
					"Africa":       {},
					"NorthAmerica": {},
				},
			},
			inRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Region: "NorthAmerica",
						},
					},
				},
			},
			expectedResult: "true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			result, err := tc.inDataCenterIn.Call(tc.inRequestWrapper)
			assert.Equal(t, tc.expectedResult, result)
			assert.Nil(t, err)
		})
	}
}

func TestChannelCall(t *testing.T) {
	testCases := []struct {
		desc                string
		inWrapper           *openrtb_ext.RequestWrapper
		expectedChannelName string
		expectedError       error
	}{
		{
			desc: "error retrieving request.ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`malformed`),
				},
			},
			expectedChannelName: "",
			expectedError:       &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc: "nil request.ext.prebid",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{}`),
				},
			},
			expectedChannelName: "",
			expectedError:       nil,
		},
		{
			desc: "nil request.ext.prebid.channel",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{}}`),
				},
			},
			expectedChannelName: "",
			expectedError:       nil,
		},
		{
			desc: "empty request.ext.prebid.name",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"channel":{}}}`),
				},
			},
			expectedChannelName: "",
			expectedError:       nil,
		},
		{
			desc: "success channel name retrieved",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"channel":{"name":"anyName"}}}`),
				},
			},
			expectedChannelName: "anyName",
			expectedError:       nil,
		},
		{
			desc: "success channel name is pbjs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"channel":{"name":"pbjs"}}}`),
				},
			},
			expectedChannelName: "web",
			expectedError:       nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			c := &channel{}

			name, err := c.Call(tc.inWrapper)
			assert.Equal(t, tc.expectedChannelName, name)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestEidAvailableCall(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    string
	}{
		{
			desc: "request.User.EIDs not found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			result: "false",
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "anySource",
							},
						},
					},
				},
			},
			result: "true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc := &eidAvailable{}

			found, err := schemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Nil(t, err)
		})
	}
}

func TestUserFpdAvailableCall(t *testing.T) {
	testCases := []struct {
		desc        string
		inWrapper   *openrtb_ext.RequestWrapper
		result      string
		expectedErr error
	}{
		{
			desc:        "nil wrapper",
			inWrapper:   nil,
			result:      "false",
			expectedErr: nil,
		},
		{
			desc: "no req.User.Data nor req.User.Ext.data[0]",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			result:      "false",
			expectedErr: nil,
		},
		{
			desc: "success req.User.Data found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Data: []openrtb2.Data{
							{ID: "foo"},
						},
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
		{
			desc: "malformed req.User.Ext.data[0]",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage(`malformed`),
					},
				},
			},
			result:      "false",
			expectedErr: &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc: "success req.User.Ext.data[0] found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage(`{"data": [{"id":"foo"}]}`),
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc := &userFpdAvailable{}

			found, err := schemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestFpdAvailableCall(t *testing.T) {
	testCases := []struct {
		desc        string
		inWrapper   *openrtb_ext.RequestWrapper
		result      string
		expectedErr error
	}{
		{
			desc:        "nil wrapper",
			inWrapper:   nil,
			result:      "false",
			expectedErr: nil,
		},
		{
			desc:        "nil bid request",
			inWrapper:   &openrtb_ext.RequestWrapper{BidRequest: nil},
			result:      "false",
			expectedErr: nil,
		},
		{
			desc: "nil site app user",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result:      "false",
			expectedErr: nil,
		},
		{
			desc: "success found wrapper.Site.Content.Data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Content: &openrtb2.Content{
							Data: []openrtb2.Data{
								{ID: "foo"},
							},
						},
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
		{
			desc: "success wrapper.Site.Ext.data[0] found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Ext: json.RawMessage(`{"data": [{"id":"foo"}]}`),
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
		{
			desc: "success found wrapper.App.Content.Data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Content: &openrtb2.Content{
							Data: []openrtb2.Data{
								{ID: "foo"},
							},
						},
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
		{
			desc: "success wrapper.App.Ext.data[0] found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Ext: json.RawMessage(`{"data": [{"id":"foo"}]}`),
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
		{
			desc: "success req.User.Data found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Data: []openrtb2.Data{
							{ID: "foo"},
						},
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
		{
			desc: "success req.User.Ext.data[0] found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage(`{"data": [{"id":"foo"}]}`),
					},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc := &fpdAvailable{}

			found, err := schemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestEidInCall(t *testing.T) {
	testCases := []struct {
		desc         string
		inWrapper    *openrtb_ext.RequestWrapper
		inSchemaFunc *eidIn
		result       string
	}{
		{
			desc: "nil request.User.EIDs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			inSchemaFunc: &eidIn{
				EidSources: []string{"fooSource", "barSource"},
				Eids: map[string]struct{}{
					"fooSource": {},
					"barSource": {},
				},
			},
			result: "false",
		},
		{
			desc: "empty request.User.EIDs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{},
					},
				},
			},
			inSchemaFunc: &eidIn{
				EidSources: []string{},
				Eids:       make(map[string]struct{}),
			},
			result: "false",
		},
		{
			desc: "request.User.EIDs not found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "anySource",
							},
						},
					},
				},
			},
			inSchemaFunc: &eidIn{
				EidSources: []string{"fooSource", "barSource"},
				Eids: map[string]struct{}{
					"fooSource": {},
					"barSource": {},
				},
			},
			result: "false",
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "anySource",
							},
						},
					},
				},
			},
			inSchemaFunc: &eidIn{
				EidSources: []string{"fooSource", "barSource", "anySource"},
				Eids: map[string]struct{}{
					"fooSource": {},
					"barSource": {},
					"anySource": {},
				},
			},
			result: "true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			found, err := tc.inSchemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Nil(t, err)
		})
	}
}

func TestGppSidInCall(t *testing.T) {
	testCases := []struct {
		desc         string
		inWrapper    *openrtb_ext.RequestWrapper
		inSchemaFunc *gppSidIn
		result       string
	}{
		{
			desc:         "empty gppSidIn.gppSids",
			inSchemaFunc: &gppSidIn{},
			result:       "false",
		},
		{
			desc: "empty request.Regs.GPPSID",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{},
					},
				},
			},
			inSchemaFunc: &gppSidIn{
				GppSids: map[int8]struct{}{
					int8(1): {},
				},
			},
			result: "false",
		},
		{
			desc: "no request.Regs.GPPSID element found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{int8(3)},
					},
				},
			},
			inSchemaFunc: &gppSidIn{
				GppSids: map[int8]struct{}{
					int8(1): {},
					int8(2): {},
				},
			},
			result: "false",
		},
		{
			desc: "Success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{int8(2)},
					},
				},
			},
			inSchemaFunc: &gppSidIn{
				GppSids: map[int8]struct{}{
					int8(1): {},
					int8(2): {},
				},
			},
			result: "true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			found, err := tc.inSchemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Nil(t, err)
		})
	}
}

func TestGppSidAvailableCall(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    string
	}{
		{
			desc: "wrapper.Regs.GPPSID not found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{},
					},
				},
			},
			result: "false",
		},
		{
			desc: "success wrapper.Regs.GPPSID found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{int8(2)},
					},
				},
			},
			result: "true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc := &gppSidAvailable{}

			result, err := schemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, result)
			assert.Nil(t, err)
		})
	}
}

func TestTcfInScopeCall(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    string
	}{
		{
			desc: "nil wrapper.Regs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result: "false",
		},
		{
			desc: "nil wrapper.Regs.GDPR",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{},
				},
			},
			result: "false",
		},
		{
			desc: "wrapper.Regs.GDPR not equal to one",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GDPR: ptrutil.ToPtr(int8(0)),
					},
				},
			},
			result: "false",
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GDPR: ptrutil.ToPtr(int8(1)),
					},
				},
			},
			result: "true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc := &tcfInScope{}

			result, err := schemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, result)
			assert.Nil(t, err)
		})
	}
}

func TestPercentCall(t *testing.T) {
	testCases := []struct {
		desc      string
		inPercent *percent
		result    string
	}{
		{
			desc: "negative-percent",
			inPercent: &percent{
				Percent: ptrutil.ToPtr(-1),
			},
			result: "false",
		},
		{
			desc: "zero-percent",
			inPercent: &percent{
				Percent: ptrutil.ToPtr(0),
			},
			result: "false",
		},
		{
			desc: "in-range-percent-above-random-value",
			inPercent: &percent{
				Percent: ptrutil.ToPtr(50),
				rand:    &mockRandomGenerator{returnValue: 49},
			},
			result: "true",
		},
		{
			desc: "in-range-percent-equals-random-value",
			inPercent: &percent{
				Percent: ptrutil.ToPtr(50),
				rand:    &mockRandomGenerator{returnValue: 50},
			},
			result: "true",
		},
		{
			desc: "in-range-percent-below-random-value",
			inPercent: &percent{
				Percent: ptrutil.ToPtr(50),
				rand:    &mockRandomGenerator{returnValue: 51},
			},
			result: "false",
		},
		{
			desc: "greater-than-one-hundred-percent",
			inPercent: &percent{
				Percent: ptrutil.ToPtr(150),
			},
			result: "true",
		},
		{
			desc: "one-hundred-percent",
			inPercent: &percent{
				Percent: ptrutil.ToPtr(100),
			},
			result: "true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := tc.inPercent.Call(nil)

			assert.Equal(t, tc.result, result)
			assert.Nil(t, err)
		})
	}
}

// mockRandomGenerator implements randomutil.RandomGenerator for testing
type mockRandomGenerator struct {
	returnValue int
}

func (g *mockRandomGenerator) Intn(n int) int {
	// Ensure return value is within the range [0,n)
	if g.returnValue >= n {
		return g.returnValue % n
	}
	return g.returnValue
}

func (g *mockRandomGenerator) GenerateInt63() int64 {
	return int64(g.returnValue)
}

func TestCheckUserDataAndUserExtData(t *testing.T) {
	testCases := []struct {
		desc          string
		inWrapper     *openrtb_ext.RequestWrapper
		result        string
		expectedError error
	}{
		{
			desc:          "nil wrapper",
			inWrapper:     nil,
			result:        "false",
			expectedError: nil,
		},
		{
			desc: "success req.User.Data found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Data: []openrtb2.Data{
							{ID: "foo"},
						},
					},
				},
			},
			result:        "true",
			expectedError: nil,
		},
		{
			desc: "no req.User",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result:        "false",
			expectedError: nil,
		},
		{
			desc: "nil req.User.Data, nil req.User.Ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			result:        "false",
			expectedError: nil,
		},
		{
			desc: "nil req.User.Data, malformed req.User.Ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage("malformed"),
					},
				},
			},
			result:        "false",
			expectedError: &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc: "nil req.User.Data, empty req.User.Ext.data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage(`{"data": []}`),
					},
				},
			},
			result:        "false",
			expectedError: nil,
		},
		{
			desc: "nil req.User.Data, req.User.Ext.data not found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage(`{}`),
					},
				},
			},
			result:        "false",
			expectedError: nil,
		},
		{
			desc: "success req.User.Ext.data[0] found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage(`{"data": [{"id":"foo"}]}`),
					},
				},
			},
			result:        "true",
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			found, err := checkUserDataAndUserExtData(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestHasUserData(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		found     bool
	}{
		{
			desc:      "nil-wrapper",
			inWrapper: nil,
			found:     false,
		},
		{
			desc: "nil-bid-request",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: nil,
			},
			found: false,
		},
		{
			desc: "nil-user",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: nil,
				},
			},
			found: false,
		},
		{
			desc: "nil-user-data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Data: nil,
					},
				},
			},
			found: false,
		},
		{
			desc: "empty-user-data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Data: []openrtb2.Data{},
					},
				},
			},
			found: false,
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Data: []openrtb2.Data{
							{ID: "foo"},
						},
					},
				},
			},
			found: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.found, hasUserData(tc.inWrapper))
		})
	}
}

func TestHasSiteContentDataOrSiteExtData(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    string
		err       error
	}{
		{
			desc:      "nil-wrapper",
			inWrapper: nil,
			result:    "false",
			err:       nil,
		},
		{
			desc: "nil-wrapper-bid-request",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: nil,
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "nil-site",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "nil-wrapper-site-content",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{},
				},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "nil-wrapper-site-content-data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Content: &openrtb2.Content{
							Data: nil,
						},
					},
				},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "empty wrapper.Site.Content.Data, nil wrapper.Site.Ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Content: &openrtb2.Content{
							Data: []openrtb2.Data{},
						},
					},
				},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "success non-empty wrapper.Site.Content.Data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Content: &openrtb2.Content{
							Data: []openrtb2.Data{
								{ID: "foo"},
							},
						},
					},
				},
			},
			result: "true",
			err:    nil,
		},
		{
			desc: "malformed wrapper.Site.Ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Ext: json.RawMessage("malformed"),
					},
				},
			},
			result: "false",
			err:    &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc: "success wrapper.Site.Ext.data[0] found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Ext: json.RawMessage(`{"data": [{"id":"foo"}]}`),
					},
				},
			},
			result: "true",
			err:    nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := hasSiteContentDataOrSiteExtData(tc.inWrapper)
			assert.Equal(t, tc.result, res)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestHasAppContentDataOrAppExtData(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    string
		err       error
	}{
		{
			desc:      "nil-wrapper",
			inWrapper: nil,
			result:    "false",
			err:       nil,
		},
		{
			desc: "nil-wrapper-bid-request",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: nil,
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "nil App",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "nil wrapper.App.Content",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{},
				},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "nil-wrapper-app-content-data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Content: &openrtb2.Content{
							Data: nil,
						},
					},
				},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "empty wrapper.App.Content.Data, nil wrapper.App.Ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Content: &openrtb2.Content{
							Data: []openrtb2.Data{},
						},
					},
				},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "success non-empty wrapper.App.Content.Data",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Content: &openrtb2.Content{
							Data: []openrtb2.Data{
								{ID: "foo"},
							},
						},
					},
				},
			},
			result: "true",
			err:    nil,
		},
		{
			desc: "malformed wrapper.App.Ext",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Ext: json.RawMessage("malformed"),
					},
				},
			},
			result: "false",
			err:    &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc: "success wrapper.App.Ext.data[0] found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Ext: json.RawMessage(`{"data": [{"id":"foo"}]}`),
					},
				},
			},
			result: "true",
			err:    nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := hasAppContentDataOrAppExtData(tc.inWrapper)
			assert.Equal(t, tc.result, res)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestExtDataPresent(t *testing.T) {
	testCases := []struct {
		desc  string
		inExt map[string]json.RawMessage
		found bool
	}{
		{
			desc:  "nil map",
			inExt: nil,
			found: false,
		},
		{
			desc:  "empty map",
			inExt: map[string]json.RawMessage{},
			found: false,
		},
		{
			desc: "data not found",
			inExt: map[string]json.RawMessage{
				"foo": json.RawMessage(`{}`),
			},
			found: false,
		},
		{
			desc: "zero-length data",
			inExt: map[string]json.RawMessage{
				"data": json.RawMessage(``),
			},
			found: false,
		},
		{
			desc: "data is not an array",
			inExt: map[string]json.RawMessage{
				"data": json.RawMessage(`{"id": "any-id"}`),
			},
			found: false,
		},
		{
			desc: "non-zero-length empty data array",
			inExt: map[string]json.RawMessage{
				"data": json.RawMessage(`[]`),
			},
			found: false,
		},
		{
			desc: "one element data array",
			inExt: map[string]json.RawMessage{
				"data": json.RawMessage(`[{}]`),
			},
			found: true,
		},
		{
			desc: "multiple elements data array",
			inExt: map[string]json.RawMessage{
				"data": json.RawMessage(`[{}, {"id": "any-id"}]`),
			},
			found: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.found, extDataPresent(tc.inExt))
		})
	}
}

func TestGetRequestRegs(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    *openrtb2.Regs
	}{
		{
			desc:      "nil wrapper",
			inWrapper: nil,
			result:    nil,
		},
		{
			desc:      "nil request",
			inWrapper: &openrtb_ext.RequestWrapper{},
			result:    nil,
		},
		{
			desc: "nil request.Regs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result: nil,
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{},
				},
			},
			result: &openrtb2.Regs{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.result, getRequestRegs(tc.inWrapper))
		})
	}
}

func TestHasGPPSIDs(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    bool
	}{
		{
			desc: "no request.Regs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result: false,
		},
		{
			desc: "empty request.Regs.GPPSID",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{},
					},
				},
			},
			result: false,
		},
		{
			desc: "nil request.Regs.GPPSID",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: nil,
					},
				},
			},
			result: false,
		},
		{
			desc: "no non-zero element found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{
							int8(0),
							int8(0),
						},
					},
				},
			},
			result: false,
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						GPPSID: []int8{
							int8(0),
							int8(1),
							int8(0),
						},
					},
				},
			},
			result: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.result, hasGPPSIDs(tc.inWrapper))
		})
	}
}

func TestNewDataCenterIn(t *testing.T) {
	testCases := []struct {
		desc                 string
		inParams             json.RawMessage
		expectedDataCenterIn SchemaFunction[openrtb_ext.RequestWrapper]
		expectedError        error
	}{
		{
			desc:                 "nil params",
			inParams:             nil,
			expectedDataCenterIn: nil,
			expectedError:        &errortypes.FailedToUnmarshal{Message: "expect { or n, but found \x00"},
		},
		{
			desc:                 "malformed params",
			inParams:             json.RawMessage(`malformed`),
			expectedDataCenterIn: nil,
			expectedError:        &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc:                 "empty params.datacenters",
			inParams:             json.RawMessage(`{"datacenters": []}`),
			expectedDataCenterIn: nil,
			expectedError:        errors.New("Empty datacenter argument in dataCenterIn schema function"),
		},
		{
			desc:                 "params.datacenters comes with non-string values",
			inParams:             json.RawMessage(`{"datacenters": [5.5, "anyDC", 1]}`),
			expectedDataCenterIn: nil,
			expectedError:        &errortypes.FailedToUnmarshal{Message: "cannot unmarshal []string: expects \" or n, but found 5"},
		},
		{
			desc:     "success",
			inParams: json.RawMessage(`{"datacenters": ["dc1"]}`),
			expectedDataCenterIn: &dataCenterIn{
				DataCenters: []string{"dc1"},
				DataCenterDir: map[string]struct{}{
					"dc1": {},
				},
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc, err := NewDataCenterIn(tc.inParams)

			assert.Equal(t, tc.expectedDataCenterIn, schemaFunc)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestNewEidIn(t *testing.T) {
	testCases := []struct {
		desc          string
		inParams      json.RawMessage
		expectedEidIn SchemaFunction[openrtb_ext.RequestWrapper]
		expectedError error
	}{
		{
			desc:          "nil params",
			inParams:      nil,
			expectedEidIn: nil,
			expectedError: &errortypes.FailedToUnmarshal{Message: "expect { or n, but found \x00"},
		},
		{
			desc:          "malformed params",
			inParams:      json.RawMessage(`malformed`),
			expectedEidIn: nil,
			expectedError: &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc:          "empty args.sources",
			inParams:      json.RawMessage(`{"sources": []}`),
			expectedEidIn: nil,
			expectedError: errors.New("Empty sources argument in eidIn schema function"),
		},
		{
			desc:          "args.sources comes with non-string values",
			inParams:      json.RawMessage(`{"sources": [5.5, "anyDC", 1]}`),
			expectedEidIn: nil,
			expectedError: &errortypes.FailedToUnmarshal{Message: "cannot unmarshal []string: expects \" or n, but found 5"},
		},
		{
			desc:     "success",
			inParams: json.RawMessage(`{"sources": ["pubcid.org"]}`),
			expectedEidIn: &eidIn{
				EidSources: []string{"pubcid.org"},
				Eids: map[string]struct{}{
					"pubcid.org": {},
				},
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc, err := NewEidIn(tc.inParams)

			assert.Equal(t, tc.expectedEidIn, schemaFunc)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestNewGppSidIn(t *testing.T) {
	testCases := []struct {
		desc             string
		inParams         json.RawMessage
		expectedGppSidIn SchemaFunction[openrtb_ext.RequestWrapper]
		expectedError    error
	}{
		{
			desc:             "nil params",
			inParams:         nil,
			expectedGppSidIn: nil,
			expectedError:    &errortypes.FailedToUnmarshal{Message: "expect { or n, but found \x00"},
		},
		{
			desc:             "malformed params",
			inParams:         json.RawMessage(`malformed`),
			expectedGppSidIn: nil,
			expectedError:    &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc:             "empty params.datacenters",
			inParams:         json.RawMessage(`{"sids": []}`),
			expectedGppSidIn: nil,
			expectedError:    errors.New("Empty GPPSIDs list argument in gppSidIn schema function"),
		},
		{
			desc:             "params.datacenters comes with non-int values",
			inParams:         json.RawMessage(`{"sids": [5.5, "anyDC", 1]}`),
			expectedGppSidIn: nil,
			expectedError:    &errortypes.FailedToUnmarshal{Message: "cannot unmarshal []int8: can not decode float as int"},
		},
		{
			desc:     "success",
			inParams: json.RawMessage(`{"sids": [1, 5]}`),
			expectedGppSidIn: &gppSidIn{
				SidList: []int8{1, 5},
				GppSids: map[int8]struct{}{
					1: {},
					5: {},
				},
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc, err := NewGppSidIn(tc.inParams)

			assert.Equal(t, tc.expectedGppSidIn, schemaFunc)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestNewPercent(t *testing.T) {
	testCases := []struct {
		desc            string
		inParams        json.RawMessage
		expectedPercent SchemaFunction[openrtb_ext.RequestWrapper]
		expectedError   error
	}{
		{
			desc:            "nil params",
			inParams:        nil,
			expectedPercent: nil,
			expectedError:   &errortypes.FailedToUnmarshal{Message: "expect { or n, but found \x00"},
		},
		{
			desc:            "malformed params",
			inParams:        json.RawMessage(`malformed`),
			expectedPercent: nil,
			expectedError:   &errortypes.FailedToUnmarshal{Message: "expect { or n, but found m"},
		},
		{
			desc:     "null pct",
			inParams: json.RawMessage(`{"pct": null}`),
			expectedPercent: &percent{
				Percent: ptrutil.ToPtr(5),
				rand:    randomutil.RandomNumberGenerator{},
			},
			expectedError: nil,
		},
		{
			desc:     "missing pct",
			inParams: json.RawMessage(`{}`),
			expectedPercent: &percent{
				Percent: ptrutil.ToPtr(5),
				rand:    randomutil.RandomNumberGenerator{},
			},
			expectedError: nil,
		},
		{
			desc:     "pct less than zero",
			inParams: json.RawMessage(`{"pct": -1}`),
			expectedPercent: &percent{
				Percent: ptrutil.ToPtr(0),
				rand:    randomutil.RandomNumberGenerator{},
			},
			expectedError: nil,
		},
		{
			desc:     "pct greater than 100",
			inParams: json.RawMessage(`{"pct": 101}`),
			expectedPercent: &percent{
				Percent: ptrutil.ToPtr(100),
				rand:    randomutil.RandomNumberGenerator{},
			},
			expectedError: nil,
		},
		{
			desc:     "pct between 0 and 100",
			inParams: json.RawMessage(`{"pct": 80}`),
			expectedPercent: &percent{
				Percent: ptrutil.ToPtr(80),
				rand:    randomutil.RandomNumberGenerator{},
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc, err := NewPercent(tc.inParams)

			assert.Equal(t, tc.expectedPercent, schemaFunc)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestConstructorsOfParamLessSchemaFunctions(t *testing.T) {
	unitTestInput := []struct {
		desc        string
		inParams    json.RawMessage
		expectError bool
	}{
		{
			desc:        "empty json object",
			inParams:    json.RawMessage(`{}`),
			expectError: false,
		},
		{
			desc:        "non-empty json object",
			inParams:    json.RawMessage(`{"someArgument": "someValue"}`),
			expectError: true,
		},
	}

	testCases := []struct {
		schemaFuncName     string
		constructorFunc    func(params json.RawMessage) (SchemaFunction[openrtb_ext.RequestWrapper], error)
		expectedSchemaFunc SchemaFunction[openrtb_ext.RequestWrapper]
	}{
		{
			schemaFuncName:     DeviceCountry,
			constructorFunc:    NewDeviceCountry,
			expectedSchemaFunc: &deviceCountry{},
		},
		{
			schemaFuncName:     DataCenter,
			constructorFunc:    NewDataCenter,
			expectedSchemaFunc: &dataCenter{},
		},
		{
			schemaFuncName:     Channel,
			constructorFunc:    NewChannel,
			expectedSchemaFunc: &channel{},
		},
		{
			schemaFuncName:     EidAvailable,
			constructorFunc:    NewEidAvailable,
			expectedSchemaFunc: &eidAvailable{},
		},
		{
			schemaFuncName:     UserFpdAvailable,
			constructorFunc:    NewUserFpdAvailable,
			expectedSchemaFunc: &userFpdAvailable{},
		},
		{
			schemaFuncName:     FpdAvailable,
			constructorFunc:    NewFpdAvailable,
			expectedSchemaFunc: &fpdAvailable{},
		},
		{
			schemaFuncName:     GppSidAvailable,
			constructorFunc:    NewGppSidAvailable,
			expectedSchemaFunc: &gppSidAvailable{},
		},
		{
			schemaFuncName:     TcfInScope,
			constructorFunc:    NewTcfInScope,
			expectedSchemaFunc: &tcfInScope{},
		},
	}

	for _, tc := range testCases {
		for _, in := range unitTestInput {
			t.Run(tc.schemaFuncName+"_"+in.desc, func(t *testing.T) {
				schemaFunc, err := tc.constructorFunc(in.inParams)

				if !in.expectError {
					assert.Equal(t, tc.expectedSchemaFunc, schemaFunc)
					assert.Nil(t, err)
				} else {
					assert.Nil(t, schemaFunc)
					assert.Equal(t, fmt.Errorf("%s expects 0 arguments", tc.schemaFuncName), err)
				}
			})
		}
	}
}

func TestSchemaFunctionsName(t *testing.T) {
	testCases := []struct {
		expectedSchemaFuncName string
		inSchemaFunc           SchemaFunction[openrtb_ext.RequestWrapper]
	}{
		{
			expectedSchemaFuncName: Channel,
			inSchemaFunc:           &channel{},
		},
		{
			expectedSchemaFuncName: DataCenter,
			inSchemaFunc:           &dataCenter{},
		},
		{
			expectedSchemaFuncName: DataCenterIn,
			inSchemaFunc:           &dataCenterIn{},
		},
		{
			expectedSchemaFuncName: DeviceCountry,
			inSchemaFunc:           &deviceCountry{},
		},
		{
			expectedSchemaFuncName: DeviceCountryIn,
			inSchemaFunc:           &deviceCountryIn{},
		},
		{
			expectedSchemaFuncName: EidAvailable,
			inSchemaFunc:           &eidAvailable{},
		},
		{
			expectedSchemaFuncName: EidIn,
			inSchemaFunc:           &eidIn{},
		},
		{
			expectedSchemaFuncName: FpdAvailable,
			inSchemaFunc:           &fpdAvailable{},
		},
		{
			expectedSchemaFuncName: GppSidAvailable,
			inSchemaFunc:           &gppSidAvailable{},
		},
		{
			expectedSchemaFuncName: GppSidIn,
			inSchemaFunc:           &gppSidIn{},
		},
		{
			expectedSchemaFuncName: Percent,
			inSchemaFunc:           &percent{},
		},
		{
			expectedSchemaFuncName: TcfInScope,
			inSchemaFunc:           &tcfInScope{},
		},
		{
			expectedSchemaFuncName: UserFpdAvailable,
			inSchemaFunc:           &userFpdAvailable{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedSchemaFuncName, func(t *testing.T) {
			assert.Equal(t, tc.expectedSchemaFuncName, tc.inSchemaFunc.Name())
		})
	}
}

func TestNewRequestSchemaFunction(t *testing.T) {
	testCases := []struct {
		inFunctionName     string
		inParams           json.RawMessage
		expectedSchemaFunc SchemaFunction[openrtb_ext.RequestWrapper]
		expectedError      error
	}{
		{
			inFunctionName:     Channel,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &channel{},
		},
		{
			inFunctionName:     DataCenter,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &dataCenter{},
		},
		{
			inFunctionName: DataCenterIn,
			inParams:       json.RawMessage(`{"datacenters": ["dc1"]}`),
			expectedSchemaFunc: &dataCenterIn{
				DataCenters: []string{"dc1"},
				DataCenterDir: map[string]struct{}{
					"dc1": {},
				},
			},
		},
		{
			inFunctionName:     DeviceCountry,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &deviceCountry{},
		},
		{
			inFunctionName: DeviceCountryIn,
			inParams:       json.RawMessage(`{"countries":["JPN"]}`),
			expectedSchemaFunc: &deviceCountryIn{
				Countries: []string{"JPN"},
				CountryDirectory: map[string]struct{}{
					"JPN": {},
				},
			},
		},
		{
			inFunctionName:     EidAvailable,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &eidAvailable{},
		},
		{
			inFunctionName: EidIn,
			inParams:       json.RawMessage(`{"sources": ["pubcid.org"]}`),
			expectedSchemaFunc: &eidIn{
				EidSources: []string{"pubcid.org"},
				Eids: map[string]struct{}{
					"pubcid.org": {},
				},
			},
		},
		{
			inFunctionName:     FpdAvailable,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &fpdAvailable{},
		},
		{
			inFunctionName:     GppSidAvailable,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &gppSidAvailable{},
		},
		{
			inFunctionName: GppSidIn,
			inParams:       json.RawMessage(`{"sids": [1, 5]}`),
			expectedSchemaFunc: &gppSidIn{
				SidList: []int8{1, 5},
				GppSids: map[int8]struct{}{
					1: {},
					5: {},
				},
			},
		},
		{
			inFunctionName: Percent,
			inParams:       json.RawMessage(`{"pct":33}`),
			expectedSchemaFunc: &percent{
				Percent: ptrutil.ToPtr(33),
				rand:    randomutil.RandomNumberGenerator{},
			},
		},
		{
			inFunctionName:     TcfInScope,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &tcfInScope{},
		},
		{
			inFunctionName:     UserFpdAvailable,
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: &userFpdAvailable{},
		},
		{
			inFunctionName:     "unknown",
			inParams:           json.RawMessage(`{}`),
			expectedSchemaFunc: nil,
			expectedError:      errors.New("Schema function unknown was not created"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.inFunctionName, func(t *testing.T) {
			schemaFunc, err := NewRequestSchemaFunction(tc.inFunctionName, tc.inParams)
			assert.Equal(t, tc.expectedSchemaFunc, schemaFunc)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestCheckNilArgs(t *testing.T) {
	testCases := []struct {
		desc          string
		inParams      json.RawMessage
		expectedError error
	}{
		{
			desc:          "malformed params",
			inParams:      json.RawMessage(`malformed`),
			expectedError: errors.New("anyFunctionName expects 0 arguments"),
		},
		{
			desc:          "non-empty json object",
			inParams:      json.RawMessage(`{"field": "value"}`),
			expectedError: errors.New("anyFunctionName expects 0 arguments"),
		},
		{
			desc:          "empty json array",
			inParams:      json.RawMessage(`[]`),
			expectedError: errors.New("anyFunctionName expects 0 arguments"),
		},
		{
			desc:          "empty json string",
			inParams:      json.RawMessage(`""`),
			expectedError: errors.New("anyFunctionName expects 0 arguments"),
		},
		{
			desc:          "nil params",
			inParams:      nil,
			expectedError: nil,
		},
		{
			desc:          "empty params",
			inParams:      json.RawMessage(``),
			expectedError: nil,
		},
		{
			desc:          "empty json object",
			inParams:      json.RawMessage(`{}`),
			expectedError: nil,
		},
		{
			desc:          "json null",
			inParams:      json.RawMessage(`null`),
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expectedError, checkNilArgs(tc.inParams, "anyFunctionName"))
		})
	}
}
