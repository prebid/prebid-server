package rules

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetDeviceGeo(t *testing.T) {
	testCases := []struct {
		desc          string
		inWrapper     *openrtb_ext.RequestWrapper
		expectedGeo   *openrtb2.Geo
		expectedError error
	}{
		{
			desc:          "nil wrapper",
			inWrapper:     nil,
			expectedGeo:   nil,
			expectedError: errors.New("request.Device.Geo is not present in request"),
		},
		{
			desc:          "nil wrapper.bidRequest",
			inWrapper:     &openrtb_ext.RequestWrapper{},
			expectedGeo:   nil,
			expectedError: errors.New("request.Device.Geo is not present in request"),
		},
		{
			desc: "nil wrapper.bidRequest.device",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectedGeo:   nil,
			expectedError: errors.New("request.Device.Geo is not present in request"),
		},
		{
			desc: "nil wrapper.bidRequest.device.geo",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
			expectedGeo:   nil,
			expectedError: errors.New("request.Device.Geo is not present in request"),
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
			expectedGeo:   &openrtb2.Geo{Country: "MEX"},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			geo, err := getDeviceGeo(tc.inWrapper)
			assert.Equal(t, tc.expectedGeo, geo)
			assert.Equal(t, tc.expectedError, err)
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
			expectedError:  errors.New("reqiuest.ext.prebid is not present in request"),
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
			expectedError:  errors.New("reqiuest.ext.prebid is not present in request"),
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
		expectedError   error
	}{
		{
			desc:            "nil wrapper",
			inWrapper:       nil,
			expectedEIDsArr: nil,
			expectedError:   errors.New("request.User.EIDs is not present in request"),
		},
		{
			desc: "nil request.User",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			expectedEIDsArr: nil,
			expectedError:   errors.New("request.User.EIDs is not present in request"),
		},
		{
			desc: "nil request.User.EIDs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			expectedEIDsArr: nil,
			expectedError:   errors.New("request.User.EIDs is not present in request"),
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
			expectedError:   errors.New("request.User.EIDs is not present in request"),
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
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			eids, err := getUserEIDS(tc.inWrapper)
			assert.Equal(t, tc.expectedEIDsArr, eids)
			assert.Equal(t, tc.expectedError, err)
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
			desc:     "array args",
			inParams: json.RawMessage(`{"countries":["JPN"]}`),
			outDeviceCountryIn: &deviceCountryIn{
				Countries: []string{"JPN"},
				CountryDirectory: map[string]struct{}{
					"JPN": struct{}{},
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

func TestDeviceCountryIn(t *testing.T) {
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
		expectedError      error
	}{
		{
			desc:               "nil wrapper.device.geo",
			inDeviceCountryIn:  deviceCountryIn{},
			inRequestWrapper:   nil,
			expectedStringBool: "false",
			expectedError:      errors.New("request.Device.Geo is not present in request"),
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
			expectedError:      errors.New("request.Device.Geo.Country is not present in request"),
		},
		{
			desc:               "empty country list",
			inDeviceCountryIn:  deviceCountryIn{},
			inRequestWrapper:   wrapperWithCountryCode,
			expectedStringBool: "false",
			expectedError:      nil,
		},
		{
			desc: "wrapper.device.geo.country not found in country list",
			inDeviceCountryIn: deviceCountryIn{
				CountryDirectory: map[string]struct{}{
					"USA": struct{}{},
					"CAN": struct{}{},
				},
			},
			inRequestWrapper:   wrapperWithCountryCode,
			expectedStringBool: "false",
		},
		{
			desc: "success",
			inDeviceCountryIn: deviceCountryIn{
				CountryDirectory: map[string]struct{}{
					"USA": struct{}{},
					"MEX": struct{}{},
					"CAN": struct{}{},
				},
			},
			inRequestWrapper:   wrapperWithCountryCode,
			expectedStringBool: "true",
			expectedError:      nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			result, err := tc.inDeviceCountryIn.Call(tc.inRequestWrapper)
			assert.Equal(t, tc.expectedStringBool, result)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestDeviceCountry(t *testing.T) {
	testCases := []struct {
		desc            string
		inWrapper       *openrtb_ext.RequestWrapper
		expectedCountry string
		expectedError   error
	}{
		{
			desc: "nil wrapper.bidRequest.device.geo",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
			expectedError: errors.New("request.Device.Geo is not present in request"),
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
			expectedError: errors.New("request.Device.Geo.Country is not present in request"),
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
			expectedError:   nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dc := &deviceCountry{}

			country, err := dc.Call(tc.inWrapper)
			assert.Equal(t, tc.expectedCountry, country)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestDataCenters(t *testing.T) {
	testCases := []struct {
		desc           string
		inWrapper      *openrtb_ext.RequestWrapper
		expectedRegion string
		expectedError  error
	}{
		{
			desc: "nil wrapper.bidRequest.device.geo",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{},
				},
			},
			expectedError: errors.New("request.Device.Geo is not present in request"),
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
			expectedError: errors.New("request.Device.Geo.Region is not present in request"),
		},
		{
			desc: "valid wrapper.bidRequest.device.geo.country",
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
			expectedError:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dc := &dataCenters{}

			region, err := dc.Call(tc.inWrapper)
			assert.Equal(t, tc.expectedRegion, region)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestDataCentersIn(t *testing.T) {
	wrapperWithRegion := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Device: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Region: "NorthAmerica",
				},
			},
		},
	}

	testCases := []struct {
		desc               string
		inDataCentersIn    dataCentersIn
		inRequestWrapper   *openrtb_ext.RequestWrapper
		expectedStringBool string
		expectedError      error
	}{
		{
			desc:               "nil wrapper.device.geo",
			inRequestWrapper:   nil,
			expectedStringBool: "false",
			expectedError:      errors.New("request.Device.Geo is not present in request"),
		},
		{
			desc:            "empty wrapper.device.geo.region",
			inDataCentersIn: dataCentersIn{},
			inRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						Geo: &openrtb2.Geo{
							Region: "",
						},
					},
				},
			},
			expectedStringBool: "false",
			expectedError:      errors.New("request.Device.Geo.Region is not present in request"),
		},
		{
			desc:               "empty region list",
			inDataCentersIn:    dataCentersIn{},
			inRequestWrapper:   wrapperWithRegion,
			expectedStringBool: "false",
			expectedError:      nil,
		},
		{
			desc: "wrapper.device.geo.region not found",
			inDataCentersIn: dataCentersIn{
				DataCenterList: []string{"Europe", "Africa"},
			},
			inRequestWrapper:   wrapperWithRegion,
			expectedStringBool: "false",
		},
		{
			desc: "success",
			inDataCentersIn: dataCentersIn{
				DataCenterList: []string{"Europe", "Africa", "NorthAmerica"},
			},
			inRequestWrapper:   wrapperWithRegion,
			expectedStringBool: "true",
			expectedError:      nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			result, err := tc.inDataCentersIn.Call(tc.inRequestWrapper)
			assert.Equal(t, tc.expectedStringBool, result)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestChannel(t *testing.T) {
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
			expectedError:       errors.New("reqiuest.ext.prebid is not present in request"),
		},
		{
			desc: "nil request.ext.prebid.channel",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{}}`),
				},
			},
			expectedChannelName: "",
			expectedError:       errors.New("reqiuest.ext.prebid or req.ext.prebid.channel is not present in request"),
		},
		{
			desc: "empty request.ext.prebid.name",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"channel":{}}}`),
				},
			},
			expectedChannelName: "",
			expectedError:       errors.New("req.ext.prebid.channel.name is not present in request"),
		},
		{
			desc: "success",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Ext: json.RawMessage(`{"prebid":{"channel":{"name":"anyName"}}}`),
				},
			},
			expectedChannelName: "anyName",
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

func TestEidAvailable(t *testing.T) {
	testCases := []struct {
		desc        string
		inWrapper   *openrtb_ext.RequestWrapper
		result      string
		expectedErr error
	}{
		{
			desc: "request.User.EIDs not found",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			result:      "false",
			expectedErr: errors.New("request.User.EIDs is not present in request"),
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
			result:      "true",
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			schemaFunc := &eidAvailable{}

			found, err := schemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestUserFpdAvailable(t *testing.T) {
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

func TestFpdAvailable(t *testing.T) {
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

func TestEidIn(t *testing.T) {
	testCases := []struct {
		desc         string
		inWrapper    *openrtb_ext.RequestWrapper
		inSchemaFunc *eidIn
		result       string
		expectedErr  error
	}{
		{
			desc: "nil request.User.EIDs",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{},
				},
			},
			inSchemaFunc: &eidIn{
				eidList: []string{"fooSource", "barSource"},
				eidDir: map[string]struct{}{
					"fooSource": struct{}{},
					"barSource": struct{}{},
				},
			},
			result:      "false",
			expectedErr: errors.New("request.User.EIDs is not present in request"),
		},
		{
			desc: "empty request.User.EIDs",
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
				eidList: []string{},
				eidDir:  make(map[string]struct{}),
			},
			result:      "false",
			expectedErr: nil,
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
				eidList: []string{"fooSource", "barSource"},
				eidDir: map[string]struct{}{
					"fooSource": struct{}{},
					"barSource": struct{}{},
				},
			},
			result:      "false",
			expectedErr: nil,
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
				eidList: []string{"fooSource", "barSource", "anySource"},
				eidDir: map[string]struct{}{
					"fooSource": struct{}{},
					"barSource": struct{}{},
					"anySource": struct{}{},
				},
			},
			result:      "true",
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			found, err := tc.inSchemaFunc.Call(tc.inWrapper)
			assert.Equal(t, tc.result, found)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

//func TestGppSidIn(t *testing.T) {
//	testCases := []struct {
//		desc         string
//		inWrapper    *openrtb_ext.RequestWrapper
//		inSchemaFunc *gppSidIn
//		result       string
//		expectedErr  error
//	}{
//		{
//			desc: "nil request.Regs",
//			inWrapper: &openrtb_ext.RequestWrapper{
//				BidRequest: &openrtb2.BidRequest{},
//			},
//			inSchemaFunc: &gppSidIn{
//				gppSids: []int8{1, 2},
//			},
//			result:      "false",
//			expectedErr: nil,
//		},
//		{
//			desc: "empty request.User.EIDs",
//			inWrapper: &openrtb_ext.RequestWrapper{
//				BidRequest: &openrtb2.BidRequest{
//					User: &openrtb2.User{
//						EIDs: []openrtb2.EID{
//							{
//								Source: "anySource",
//							},
//						},
//					},
//				},
//			},
//			inSchemaFunc: &gppSidIn{
//				eids: []string{},
//			},
//			result:      "false",
//			expectedErr: nil,
//		},
//		{
//			desc: "request.User.EIDs not found",
//			inWrapper: &openrtb_ext.RequestWrapper{
//				BidRequest: &openrtb2.BidRequest{
//					User: &openrtb2.User{
//						EIDs: []openrtb2.EID{
//							{
//								Source: "anySource",
//							},
//						},
//					},
//				},
//			},
//			inSchemaFunc: &gppSidIn{
//				eids: []string{"fooSource", "barSource"},
//			},
//			result:      "false",
//			expectedErr: nil,
//		},
//		{
//			desc: "success",
//			inWrapper: &openrtb_ext.RequestWrapper{
//				BidRequest: &openrtb2.BidRequest{
//					User: &openrtb2.User{
//						EIDs: []openrtb2.EID{
//							{
//								Source: "anySource",
//							},
//						},
//					},
//				},
//			},
//			inSchemaFunc: &gppSidIn{
//				eids: []string{"fooSource", "barSource", "anySource"},
//			},
//			result:      "true",
//			expectedErr: nil,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.desc, func(t *testing.T) {
//			found, err := tc.inSchemaFunc.Call(tc.inWrapper)
//			assert.Equal(t, tc.result, found)
//			assert.Equal(t, tc.expectedErr, err)
//		})
//	}
//}

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
			desc:      "nil wrapper",
			inWrapper: nil,
			found:     false,
		},
		{
			desc:      "nil wrapper.BidRequest",
			inWrapper: &openrtb_ext.RequestWrapper{},
			found:     false,
		},
		{
			desc: "nil wrapper.User",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
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

func TestHasSiteContentData(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    string
		err       error
	}{
		{
			desc: "nil Site",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			result: "false",
			err:    nil,
		},
		{
			desc: "nil wrapper.Site.Content",
			inWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{},
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
			res, err := checkSiteContentDataAndSiteExtData(tc.inWrapper)
			assert.Equal(t, tc.result, res)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestHasAppContentData(t *testing.T) {
	testCases := []struct {
		desc      string
		inWrapper *openrtb_ext.RequestWrapper
		result    string
		err       error
	}{
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
			res, err := checkAppContentDataAndAppExtData(tc.inWrapper)
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
			desc: "zero-lenght data",
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
			desc: "non-zero-lenght empty data array",
			inExt: map[string]json.RawMessage{
				"data": json.RawMessage(`[]`),
			},
			found: false,
		},
		{
			desc: "success",
			inExt: map[string]json.RawMessage{
				"data": json.RawMessage(`[{"id": "any-id"}]`),
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
