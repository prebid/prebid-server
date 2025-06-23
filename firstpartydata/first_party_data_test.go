package firstpartydata

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractGlobalFPD(t *testing.T) {
	testCases := []struct {
		description string
		input       openrtb_ext.RequestWrapper
		expectedReq openrtb_ext.RequestWrapper
		expectedFpd map[string][]byte
	}{
		{
			description: "Site, app and user data present",
			input: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
						Ext: json.RawMessage(`{"data":{"somesitefpd":"sitefpdDataTest"}}`),
					},
					User: &openrtb2.User{
						ID:     "reqUserID",
						Yob:    1982,
						Gender: "M",
						Ext:    json.RawMessage(`{"data":{"someuserfpd":"userfpdDataTest"}}`),
					},
					App: &openrtb2.App{
						ID:  "appId",
						Ext: json.RawMessage(`{"data":{"someappfpd":"appfpdDataTest"}}`),
					},
				},
			},
			expectedReq: openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				ID: "bid_id",
				Site: &openrtb2.Site{
					ID:   "reqSiteId",
					Page: "http://www.foobar.com/1234.html",
					Publisher: &openrtb2.Publisher{
						ID: "1",
					},
				},
				User: &openrtb2.User{
					ID:     "reqUserID",
					Yob:    1982,
					Gender: "M",
				},
				App: &openrtb2.App{
					ID: "appId",
				},
			}},
			expectedFpd: map[string][]byte{
				"site": []byte(`{"somesitefpd":"sitefpdDataTest"}`),
				"user": []byte(`{"someuserfpd":"userfpdDataTest"}`),
				"app":  []byte(`{"someappfpd":"appfpdDataTest"}`),
			},
		},
		{
			description: "App FPD only present",
			input: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
					},
					App: &openrtb2.App{
						ID:  "appId",
						Ext: json.RawMessage(`{"data":{"someappfpd":"appfpdDataTest"}}`),
					},
				},
			},
			expectedReq: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
					},
					App: &openrtb2.App{
						ID: "appId",
					},
				},
			},
			expectedFpd: map[string][]byte{
				"app":  []byte(`{"someappfpd":"appfpdDataTest"}`),
				"user": nil,
				"site": nil,
			},
		},
		{
			description: "User FPD only present",
			input: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
					},
					User: &openrtb2.User{
						ID:     "reqUserID",
						Yob:    1982,
						Gender: "M",
						Ext:    json.RawMessage(`{"data":{"someuserfpd":"userfpdDataTest"}}`),
					},
				},
			},
			expectedReq: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
					},
					User: &openrtb2.User{
						ID:     "reqUserID",
						Yob:    1982,
						Gender: "M",
					},
				},
			},
			expectedFpd: map[string][]byte{
				"app":  nil,
				"user": []byte(`{"someuserfpd":"userfpdDataTest"}`),
				"site": nil,
			},
		},
		{
			description: "No FPD present in req",
			input: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
					},
					User: &openrtb2.User{
						ID:     "reqUserID",
						Yob:    1982,
						Gender: "M",
					},
					App: &openrtb2.App{
						ID: "appId",
					},
				},
			},
			expectedReq: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
					},
					User: &openrtb2.User{
						ID:     "reqUserID",
						Yob:    1982,
						Gender: "M",
					},
					App: &openrtb2.App{
						ID: "appId",
					},
				},
			},
			expectedFpd: map[string][]byte{
				"app":  nil,
				"user": nil,
				"site": nil,
			},
		},
		{
			description: "Site FPD only present",
			input: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
						Ext: json.RawMessage(`{"data":{"someappfpd":true}}`),
					},
					App: &openrtb2.App{
						ID: "appId",
					},
				},
			},
			expectedReq: openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "bid_id",
					Site: &openrtb2.Site{
						ID:   "reqSiteId",
						Page: "http://www.foobar.com/1234.html",
						Publisher: &openrtb2.Publisher{
							ID: "1",
						},
					},
					App: &openrtb2.App{
						ID: "appId",
					},
				},
			},
			expectedFpd: map[string][]byte{
				"app":  nil,
				"user": nil,
				"site": []byte(`{"someappfpd":true}`),
			},
		},
	}
	for _, test := range testCases {

		inputReq := &test.input
		fpd, err := ExtractGlobalFPD(inputReq)
		assert.NoError(t, err, "Error should be nil")
		err = inputReq.RebuildRequest()
		assert.NoError(t, err, "Error should be nil")

		assert.Equal(t, test.expectedReq.BidRequest, inputReq.BidRequest, "Incorrect input request after global fpd extraction")

		assert.Equal(t, test.expectedFpd[userKey], fpd[userKey], "Incorrect User FPD")
		assert.Equal(t, test.expectedFpd[appKey], fpd[appKey], "Incorrect App FPD")
		assert.Equal(t, test.expectedFpd[siteKey], fpd[siteKey], "Incorrect Site FPDt")
	}
}

func TestExtractOpenRtbGlobalFPD(t *testing.T) {
	testCases := []struct {
		description     string
		input           openrtb2.BidRequest
		output          openrtb2.BidRequest
		expectedFpdData map[string][]openrtb2.Data
	}{
		{
			description: "Site, app and user data present",
			input: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				Site: &openrtb2.Site{
					ID: "reqSiteId",
					Content: &openrtb2.Content{
						Data: []openrtb2.Data{
							{ID: "siteDataId1", Name: "siteDataName1"},
							{ID: "siteDataId2", Name: "siteDataName2"},
						},
					},
				},
				User: &openrtb2.User{
					ID:     "reqUserID",
					Yob:    1982,
					Gender: "M",
					Data: []openrtb2.Data{
						{ID: "userDataId1", Name: "userDataName1"},
					},
				},
				App: &openrtb2.App{
					ID: "appId",
					Content: &openrtb2.Content{
						Data: []openrtb2.Data{
							{ID: "appDataId1", Name: "appDataName1"},
						},
					},
				},
			},
			output: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				Site: &openrtb2.Site{
					ID:      "reqSiteId",
					Content: &openrtb2.Content{},
				},
				User: &openrtb2.User{
					ID:     "reqUserID",
					Yob:    1982,
					Gender: "M",
				},
				App: &openrtb2.App{
					ID:      "appId",
					Content: &openrtb2.Content{},
				},
			},
			expectedFpdData: map[string][]openrtb2.Data{
				siteContentDataKey: {{ID: "siteDataId1", Name: "siteDataName1"}, {ID: "siteDataId2", Name: "siteDataName2"}},
				userDataKey:        {{ID: "userDataId1", Name: "userDataName1"}},
				appContentDataKey:  {{ID: "appDataId1", Name: "appDataName1"}},
			},
		},
		{
			description: "No Site, app or user data present",
			input: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
			},
			output: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
			},
			expectedFpdData: map[string][]openrtb2.Data{
				siteContentDataKey: nil,
				userDataKey:        nil,
				appContentDataKey:  nil,
			},
		},
		{
			description: "Site only data present",
			input: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				Site: &openrtb2.Site{
					ID:   "reqSiteId",
					Page: "test/page",
					Content: &openrtb2.Content{
						Data: []openrtb2.Data{
							{ID: "siteDataId1", Name: "siteDataName1"},
						},
					},
				},
			},
			output: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				Site: &openrtb2.Site{
					ID:      "reqSiteId",
					Page:    "test/page",
					Content: &openrtb2.Content{},
				},
			},
			expectedFpdData: map[string][]openrtb2.Data{
				siteContentDataKey: {{ID: "siteDataId1", Name: "siteDataName1"}},
				userDataKey:        nil,
				appContentDataKey:  nil,
			},
		},
		{
			description: "App only data present",
			input: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				App: &openrtb2.App{
					ID: "reqAppId",
					Content: &openrtb2.Content{
						Data: []openrtb2.Data{
							{ID: "appDataId1", Name: "appDataName1"},
						},
					},
				},
			},
			output: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				App: &openrtb2.App{
					ID:      "reqAppId",
					Content: &openrtb2.Content{},
				},
			},
			expectedFpdData: map[string][]openrtb2.Data{
				siteContentDataKey: nil,
				userDataKey:        nil,
				appContentDataKey:  {{ID: "appDataId1", Name: "appDataName1"}},
			},
		},
		{
			description: "User only data present",
			input: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				Site: &openrtb2.Site{
					ID: "reqSiteId",
				},
				App: &openrtb2.App{
					ID: "reqAppId",
				},
				User: &openrtb2.User{
					ID:     "reqUserId",
					Yob:    1982,
					Gender: "M",
					Data: []openrtb2.Data{
						{ID: "userDataId1", Name: "userDataName1"},
					},
				},
			},
			output: openrtb2.BidRequest{
				ID: "bid_id",
				Imp: []openrtb2.Imp{
					{ID: "impid"},
				},
				Site: &openrtb2.Site{
					ID: "reqSiteId",
				},
				App: &openrtb2.App{
					ID: "reqAppId",
				},
				User: &openrtb2.User{
					ID:     "reqUserId",
					Yob:    1982,
					Gender: "M",
				},
			},
			expectedFpdData: map[string][]openrtb2.Data{
				siteContentDataKey: nil,
				userDataKey:        {{ID: "userDataId1", Name: "userDataName1"}},
				appContentDataKey:  nil,
			},
		},
	}
	for _, test := range testCases {

		inputReq := &test.input

		res := ExtractOpenRtbGlobalFPD(inputReq)

		assert.Equal(t, &test.output, inputReq, "Result request is incorrect")
		assert.Equal(t, test.expectedFpdData[siteContentDataKey], res[siteContentDataKey], "siteContentData data is incorrect")
		assert.Equal(t, test.expectedFpdData[userDataKey], res[userDataKey], "userData is incorrect")
		assert.Equal(t, test.expectedFpdData[appContentDataKey], res[appContentDataKey], "appContentData is incorrect")

	}
}

func TestExtractBidderConfigFPD(t *testing.T) {
	testPath := "tests/extractbidderconfigfpd"

	tests, err := os.ReadDir(testPath)
	require.NoError(t, err, "Cannot Discover Tests")

	for _, test := range tests {
		t.Run(test.Name(), func(t *testing.T) {
			path := filepath.Join(testPath, test.Name())

			testFile, err := loadTestFile[fpdFile](path)
			require.NoError(t, err, "Load Test File")

			givenRequestExtPrebid := &openrtb_ext.ExtRequestPrebid{}
			err = jsonutil.UnmarshalValid(testFile.InputRequestData, givenRequestExtPrebid)
			require.NoError(t, err, "Cannot Load Test Conditions")

			testRequest := &openrtb_ext.RequestExt{}
			testRequest.SetPrebid(givenRequestExtPrebid)

			// run test
			results, err := ExtractBidderConfigFPD(testRequest)

			// assert errors
			if len(testFile.ValidationErrors) > 0 {
				require.EqualError(t, err, testFile.ValidationErrors[0].Message, "Expected Error Not Received")
			} else {
				require.NoError(t, err, "Error Not Expected")
				assert.Nil(t, testRequest.GetPrebid().BidderConfigs, "Bidder specific FPD config should be removed from request")
			}

			// assert fpd (with normalization for nicer looking tests)
			for bidderName, expectedFPD := range testFile.BidderConfigFPD {
				require.Contains(t, results, bidderName)

				if expectedFPD.App != nil {
					assert.JSONEq(t, string(expectedFPD.App), string(results[bidderName].App), "app is incorrect")
				} else {
					assert.Nil(t, results[bidderName].App, "app expected to be nil")
				}

				if expectedFPD.Site != nil {
					assert.JSONEq(t, string(expectedFPD.Site), string(results[bidderName].Site), "site is incorrect")
				} else {
					assert.Nil(t, results[bidderName].Site, "site expected to be nil")
				}

				if expectedFPD.User != nil {
					assert.JSONEq(t, string(expectedFPD.User), string(results[bidderName].User), "user is incorrect")
				} else {
					assert.Nil(t, results[bidderName].User, "user expected to be nil")
				}

				if expectedFPD.Device != nil {
					assert.JSONEq(t, string(expectedFPD.Device), string(results[bidderName].Device), "device is incorrect")
				} else {
					assert.Nil(t, results[bidderName].Device, "device expected to be nil")
				}
			}
		})
	}
}

func TestResolveFPD(t *testing.T) {
	testPath := "tests/resolvefpd"

	tests, err := os.ReadDir(testPath)
	require.NoError(t, err, "Cannot Discover Tests")

	for _, test := range tests {
		t.Run(test.Name(), func(t *testing.T) {
			path := filepath.Join(testPath, test.Name())

			testFile, err := loadTestFile[fpdFileForResolveFPD](path)
			require.NoError(t, err, "Load Test File")

			request := &openrtb2.BidRequest{}
			err = jsonutil.UnmarshalValid(testFile.InputRequestData, &request)
			require.NoError(t, err, "Cannot Load Request")

			originalRequest := &openrtb2.BidRequest{}
			err = jsonutil.UnmarshalValid(testFile.InputRequestData, &originalRequest)
			require.NoError(t, err, "Cannot Load Request")

			reqExtFPD := make(map[string][]byte)
			reqExtFPD["site"] = testFile.GlobalFPD["site"]
			reqExtFPD["app"] = testFile.GlobalFPD["app"]
			reqExtFPD["user"] = testFile.GlobalFPD["user"]

			reqFPD := make(map[string][]openrtb2.Data, 3)

			reqFPDSiteContentData := testFile.GlobalFPD[siteContentDataKey]
			if len(reqFPDSiteContentData) > 0 {
				var siteConData []openrtb2.Data
				err = jsonutil.UnmarshalValid(reqFPDSiteContentData, &siteConData)
				if err != nil {
					t.Errorf("Unable to unmarshal site.content.data:")
				}
				reqFPD[siteContentDataKey] = siteConData
			}

			reqFPDAppContentData := testFile.GlobalFPD[appContentDataKey]
			if len(reqFPDAppContentData) > 0 {
				var appConData []openrtb2.Data
				err = jsonutil.UnmarshalValid(reqFPDAppContentData, &appConData)
				if err != nil {
					t.Errorf("Unable to unmarshal app.content.data: ")
				}
				reqFPD[appContentDataKey] = appConData
			}

			reqFPDUserData := testFile.GlobalFPD[userDataKey]
			if len(reqFPDUserData) > 0 {
				var userData []openrtb2.Data
				err = jsonutil.UnmarshalValid(reqFPDUserData, &userData)
				if err != nil {
					t.Errorf("Unable to unmarshal app.content.data: ")
				}
				reqFPD[userDataKey] = userData
			}

			// run test
			resultFPD, errL := ResolveFPD(request, testFile.BidderConfigFPD, reqExtFPD, reqFPD, testFile.BiddersWithGlobalFPD)

			if len(errL) == 0 {
				assert.Equal(t, request, originalRequest, "Original request should not be modified")

				expectedResultKeys := []string{}
				for k := range testFile.OutputRequestData {
					expectedResultKeys = append(expectedResultKeys, k.String())
				}
				actualResultKeys := []string{}
				for k := range resultFPD {
					actualResultKeys = append(actualResultKeys, k.String())
				}
				require.ElementsMatch(t, expectedResultKeys, actualResultKeys)

				for k, outputReq := range testFile.OutputRequestData {
					bidderFPD := resultFPD[k]

					if outputReq.Site != nil && len(outputReq.Site.Ext) > 0 {
						resSiteExt := bidderFPD.Site.Ext
						expectedSiteExt := outputReq.Site.Ext
						bidderFPD.Site.Ext = nil
						outputReq.Site.Ext = nil
						assert.JSONEq(t, string(expectedSiteExt), string(resSiteExt), "site.ext is incorrect")
						assert.Equal(t, outputReq.Site, bidderFPD.Site, "Site is incorrect")
					}
					if outputReq.App != nil && len(outputReq.App.Ext) > 0 {
						resAppExt := bidderFPD.App.Ext
						expectedAppExt := outputReq.App.Ext
						bidderFPD.App.Ext = nil
						outputReq.App.Ext = nil
						assert.JSONEq(t, string(expectedAppExt), string(resAppExt), "app.ext is incorrect")
						assert.Equal(t, outputReq.App, bidderFPD.App, "App is incorrect")
					}
					if outputReq.User != nil && len(outputReq.User.Ext) > 0 {
						resUserExt := bidderFPD.User.Ext
						expectedUserExt := outputReq.User.Ext
						bidderFPD.User.Ext = nil
						outputReq.User.Ext = nil
						assert.JSONEq(t, string(expectedUserExt), string(resUserExt), "user.ext is incorrect")
						assert.Equal(t, outputReq.User, bidderFPD.User, "User is incorrect")
					}
					if outputReq.Device != nil && len(outputReq.Device.Ext) > 0 {
						resDeviceExt := bidderFPD.Device.Ext
						expectedDeviceExt := outputReq.Device.Ext
						bidderFPD.Device.Ext = nil
						outputReq.Device.Ext = nil
						assert.JSONEq(t, string(expectedDeviceExt), string(resDeviceExt), "device.ext is incorrect")
						assert.Equal(t, outputReq.Device, bidderFPD.Device, "Device is incorrect")
					}
				}
			} else {
				assert.ElementsMatch(t, errL, testFile.ValidationErrors, "Incorrect first party data warning message")
			}
		})
	}
}

func TestExtractFPDForBidders(t *testing.T) {
	if specFiles, err := os.ReadDir("./tests/extractfpdforbidders"); err == nil {
		for _, specFile := range specFiles {
			path := filepath.Join("./tests/extractfpdforbidders/", specFile.Name())

			testFile, err := loadTestFile[fpdFile](path)
			require.NoError(t, err, "Load Test File")

			var expectedRequest openrtb2.BidRequest
			err = jsonutil.UnmarshalValid(testFile.OutputRequestData, &expectedRequest)
			if err != nil {
				t.Errorf("Unable to unmarshal input request: %s", path)
			}

			resultRequest := &openrtb_ext.RequestWrapper{}
			resultRequest.BidRequest = &openrtb2.BidRequest{}
			err = jsonutil.UnmarshalValid(testFile.InputRequestData, resultRequest.BidRequest)
			assert.NoError(t, err, "Error should be nil")

			resultFPD, errL := ExtractFPDForBidders(resultRequest)

			if len(testFile.ValidationErrors) > 0 {
				assert.Equal(t, len(testFile.ValidationErrors), len(errL), "Incorrect number of errors was returned")
				assert.ElementsMatch(t, errL, testFile.ValidationErrors, "Incorrect errors were returned")
				//in case or error no further assertions needed
				continue
			}
			assert.Empty(t, errL, "Error should be empty")
			assert.Equal(t, len(resultFPD), len(testFile.BiddersFPDResolved))

			for bidderName, expectedValue := range testFile.BiddersFPDResolved {
				actualValue := resultFPD[bidderName]
				if expectedValue.Site != nil {
					if len(expectedValue.Site.Ext) > 0 {
						assert.JSONEq(t, string(expectedValue.Site.Ext), string(actualValue.Site.Ext), "Incorrect first party data")
						expectedValue.Site.Ext = nil
						actualValue.Site.Ext = nil
					}
					assert.Equal(t, expectedValue.Site, actualValue.Site, "Incorrect first party data")
				}
				if expectedValue.App != nil {
					if len(expectedValue.App.Ext) > 0 {
						assert.JSONEq(t, string(expectedValue.App.Ext), string(actualValue.App.Ext), "Incorrect first party data")
						expectedValue.App.Ext = nil
						actualValue.App.Ext = nil
					}
					assert.Equal(t, expectedValue.App, actualValue.App, "Incorrect first party data")
				}
				if expectedValue.User != nil {
					if len(expectedValue.User.Ext) > 0 {
						assert.JSONEq(t, string(expectedValue.User.Ext), string(actualValue.User.Ext), "Incorrect first party data")
						expectedValue.User.Ext = nil
						actualValue.User.Ext = nil
					}
					assert.Equal(t, expectedValue.User, actualValue.User, "Incorrect first party data")
				}
			}

			if expectedRequest.Site != nil {
				if len(expectedRequest.Site.Ext) > 0 {
					assert.JSONEq(t, string(expectedRequest.Site.Ext), string(resultRequest.BidRequest.Site.Ext), "Incorrect site in request")
					expectedRequest.Site.Ext = nil
					resultRequest.BidRequest.Site.Ext = nil
				}
				assert.Equal(t, expectedRequest.Site, resultRequest.BidRequest.Site, "Incorrect site in request")
			}
			if expectedRequest.App != nil {
				if len(expectedRequest.App.Ext) > 0 {
					assert.JSONEq(t, string(expectedRequest.App.Ext), string(resultRequest.BidRequest.App.Ext), "Incorrect app in request")
					expectedRequest.App.Ext = nil
					resultRequest.BidRequest.App.Ext = nil
				}
				assert.Equal(t, expectedRequest.App, resultRequest.BidRequest.App, "Incorrect app in request")
			}
			if expectedRequest.User != nil {
				if len(expectedRequest.User.Ext) > 0 {
					assert.JSONEq(t, string(expectedRequest.User.Ext), string(resultRequest.BidRequest.User.Ext), "Incorrect user in request")
					expectedRequest.User.Ext = nil
					resultRequest.BidRequest.User.Ext = nil
				}
				assert.Equal(t, expectedRequest.User, resultRequest.BidRequest.User, "Incorrect user in request")
			}
		}
	}
}

func TestResolveUser(t *testing.T) {
	testCases := []struct {
		description      string
		fpdConfig        *openrtb_ext.ORTB2
		bidRequestUser   *openrtb2.User
		globalFPD        map[string][]byte
		openRtbGlobalFPD map[string][]openrtb2.Data
		expectedUser     *openrtb2.User
		expectError      string
	}{
		{
			description:  "FPD config and bid request user are not specified",
			expectedUser: nil,
		},
		{
			description:  "FPD config user only is specified",
			fpdConfig:    &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test"}`)},
			expectedUser: &openrtb2.User{ID: "test"},
		},
		{
			description:    "FPD config and bid request user are specified",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2"},
			expectedUser:   &openrtb2.User{ID: "test1"},
		},
		{
			description:    "FPD config, bid request and global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2"},
			globalFPD:      map[string][]byte{userKey: []byte(`{"globalFPDUserData":"globalFPDUserDataValue"}`)},
			expectedUser:   &openrtb2.User{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDUserData":"globalFPDUserDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request user with ext and global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"test":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
			globalFPD:      map[string][]byte{userKey: []byte(`{"globalFPDUserData":"globalFPDUserDataValue"}`)},
			expectedUser:   &openrtb2.User{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDUserData":"globalFPDUserDataValue"},"test":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd user are specified, with input user ext.data",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
			globalFPD:      map[string][]byte{userKey: []byte(`{"globalFPDUserData":"globalFPDUserDataValue"}`)},
			expectedUser:   &openrtb2.User{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDUserData":"globalFPDUserDataValue","inputFPDUserData":"inputFPDUserDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd user are specified, with input user ext.data malformed",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
			globalFPD:      map[string][]byte{userKey: []byte(`malformed`)},
			expectError:    "invalid first party data ext",
		},
		{
			description:    "bid request and openrtb global fpd user are specified, no input user ext",
			bidRequestUser: &openrtb2.User{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{userDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedUser: &openrtb2.User{ID: "test2", Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
		},
		{
			description:    "fpd config user, bid request and openrtb global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{userDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedUser: &openrtb2.User{ID: "test1", Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
		},
		{
			description:    "fpd config user with ext, bid request and openrtb global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test1","ext":{"test":1}}`)},
			bidRequestUser: &openrtb2.User{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{userDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedUser: &openrtb2.User{ID: "test1", Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			},
				Ext: json.RawMessage(`{"test":1}`)},
		},
		{
			description:    "fpd config user with ext, bid requestuser with ext and openrtb global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id":"test1","ext":{"test":1}}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"test":2,"key":"value"}`)},
			openRtbGlobalFPD: map[string][]openrtb2.Data{userDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedUser: &openrtb2.User{ID: "test1", Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			},
				Ext: json.RawMessage(`{"key":"value","test":1}`)},
		},
		{
			description:    "fpd config user with malformed ext, bid requestuser with ext and openrtb global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1","ext":{malformed}}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"test":2,"key":"value"}`)},
			openRtbGlobalFPD: map[string][]openrtb2.Data{userDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedUser: &openrtb2.User{ID: "test1", Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			},
				Ext: json.RawMessage(`{"key":"value","test":1}`),
			},
			expectError: "invalid first party data ext",
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resultUser, err := resolveUser(test.fpdConfig, test.bidRequestUser, test.globalFPD, test.openRtbGlobalFPD, "bidderA")

			if len(test.expectError) > 0 {
				assert.EqualError(t, err, test.expectError)
			} else {
				assert.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedUser, resultUser, "Result user is incorrect")
			}
		})
	}
}

func TestResolveSite(t *testing.T) {
	testCases := []struct {
		description      string
		fpdConfig        *openrtb_ext.ORTB2
		bidRequestSite   *openrtb2.Site
		globalFPD        map[string][]byte
		openRtbGlobalFPD map[string][]openrtb2.Data
		expectedSite     *openrtb2.Site
		expectError      string
	}{
		{
			description:  "FPD config and bid request site are not specified",
			expectedSite: nil,
		},
		{
			description: "FPD config site only is specified",
			fpdConfig:   &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test"}`)},
			expectError: "incorrect First Party Data for bidder bidderA: Site object is not defined in request, but defined in FPD config",
		},
		{
			description:    "FPD config and bid request site are specified",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2"},
			expectedSite:   &openrtb2.Site{ID: "test1"},
		},
		{
			description:    "FPD config, bid request and global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2"},
			globalFPD:      map[string][]byte{siteKey: []byte(`{"globalFPDSiteData":"globalFPDSiteDataValue"}`)},
			expectedSite:   &openrtb2.Site{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDSiteData":"globalFPDSiteDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request site with ext and global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"test":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
			globalFPD:      map[string][]byte{siteKey: []byte(`{"globalFPDSiteData":"globalFPDSiteDataValue"}`)},
			expectedSite:   &openrtb2.Site{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDSiteData":"globalFPDSiteDataValue"},"test":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd site are specified, with input site ext.data",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
			globalFPD:      map[string][]byte{siteKey: []byte(`{"globalFPDSiteData":"globalFPDSiteDataValue"}`)},
			expectedSite:   &openrtb2.Site{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDSiteData":"globalFPDSiteDataValue","inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd site are specified, with input site ext.data malformed",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
			globalFPD:      map[string][]byte{siteKey: []byte(`malformed`)},
			expectError:    "invalid first party data ext",
		},
		{
			description:    "bid request and openrtb global fpd site are specified, no input site ext",
			bidRequestSite: &openrtb2.Site{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{siteContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedSite: &openrtb2.Site{ID: "test2", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}}},
		},
		{
			description: "bid request with content and openrtb global fpd site are specified, no input site ext",
			bidRequestSite: &openrtb2.Site{ID: "test2", Content: &openrtb2.Content{
				ID: "InputSiteContentId",
				Data: []openrtb2.Data{
					{ID: "1", Name: "N1"},
					{ID: "2", Name: "N2"},
				},
				Ext: json.RawMessage(`{"contentPresent":true}`),
			}},
			openRtbGlobalFPD: map[string][]openrtb2.Data{siteContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedSite: &openrtb2.Site{ID: "test2", Content: &openrtb2.Content{
				ID: "InputSiteContentId",
				Data: []openrtb2.Data{
					{ID: "DataId1", Name: "Name1"},
					{ID: "DataId2", Name: "Name2"},
				},
				Ext: json.RawMessage(`{"contentPresent":true}`),
			}},
		},
		{
			description:    "fpd config site, bid request and openrtb global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{siteContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedSite: &openrtb2.Site{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}}},
		},
		{
			description:    "fpd config site with ext, bid request and openrtb global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1","ext":{"test":1}}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{siteContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedSite: &openrtb2.Site{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
				Ext: json.RawMessage(`{"test":1}`)},
		},
		{
			description:    "fpd config site with ext, bid request site with ext and openrtb global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1","ext":{"test":1}}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"test":2,"key":"value"}`)},
			openRtbGlobalFPD: map[string][]openrtb2.Data{siteContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedSite: &openrtb2.Site{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
				Ext: json.RawMessage(`{"key":"value","test":1}`)},
		},
		{
			description:    "fpd config site with malformed ext, bid request site with ext and openrtb global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"test1","ext":{malformed}}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"test":2,"key":"value"}`)},
			openRtbGlobalFPD: map[string][]openrtb2.Data{siteContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedSite: &openrtb2.Site{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
				Ext: json.RawMessage(`{"key":"value","test":1}`),
			},
			expectError: "invalid first party data ext",
		},
		{
			description:    "valid-id",
			bidRequestSite: &openrtb2.Site{ID: "1"},
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":"2"}`)},
			expectedSite:   &openrtb2.Site{ID: "2"},
		},
		{
			description:    "valid-page",
			bidRequestSite: &openrtb2.Site{Page: "1"},
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"page":"2"}`)},
			expectedSite:   &openrtb2.Site{Page: "2"},
		},
		{
			description:    "invalid-id",
			bidRequestSite: &openrtb2.Site{ID: "1"},
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id":null}`)},
			expectError:    "incorrect First Party Data for bidder bidderA: Site object cannot set empty page if req.site.id is empty",
		},
		{
			description:    "invalid-page",
			bidRequestSite: &openrtb2.Site{Page: "1"},
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"page":null}`)},
			expectError:    "incorrect First Party Data for bidder bidderA: Site object cannot set empty page if req.site.id is empty",
		},
		{
			description:    "existing-err",
			bidRequestSite: &openrtb2.Site{ID: "1", Ext: []byte(`malformed`)},
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"ext":{"a":1}}`)},
			expectError:    "invalid request ext",
		},
		{
			description:    "fpd-err",
			bidRequestSite: &openrtb2.Site{ID: "1", Ext: []byte(`{"a":1}`)},
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`malformed`)},
			expectError:    "invalid first party data ext",
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resultSite, err := resolveSite(test.fpdConfig, test.bidRequestSite, test.globalFPD, test.openRtbGlobalFPD, "bidderA")

			if len(test.expectError) > 0 {
				assert.EqualError(t, err, test.expectError)
			} else {
				require.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedSite, resultSite, "Result site is incorrect")
			}
		})
	}
}

func TestResolveApp(t *testing.T) {
	testCases := []struct {
		description      string
		fpdConfig        *openrtb_ext.ORTB2
		bidRequestApp    *openrtb2.App
		globalFPD        map[string][]byte
		openRtbGlobalFPD map[string][]openrtb2.Data
		expectedApp      *openrtb2.App
		expectError      string
	}{
		{
			description: "FPD config and bid request app are not specified",
			expectedApp: nil,
		},
		{
			description: "FPD config app only is specified",
			fpdConfig:   &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test"}`)},
			expectError: "incorrect First Party Data for bidder bidderA: App object is not defined in request, but defined in FPD config",
		},
		{
			description:   "FPD config and bid request app are specified",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2"},
			expectedApp:   &openrtb2.App{ID: "test1"},
		},
		{
			description:   "FPD config, bid request and global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2"},
			globalFPD:     map[string][]byte{appKey: []byte(`{"globalFPDAppData":"globalFPDAppDataValue"}`)},
			expectedApp:   &openrtb2.App{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDAppData":"globalFPDAppDataValue"}}`)},
		},
		{
			description:   "FPD config, bid request app with ext and global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"test":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
			globalFPD:     map[string][]byte{appKey: []byte(`{"globalFPDAppData":"globalFPDAppDataValue"}`)},
			expectedApp:   &openrtb2.App{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDAppData":"globalFPDAppDataValue"},"test":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
		},
		{
			description:   "FPD config, bid request and global fpd app are specified, with input app ext.data",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
			globalFPD:     map[string][]byte{appKey: []byte(`{"globalFPDAppData":"globalFPDAppDataValue"}`)},
			expectedApp:   &openrtb2.App{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDAppData":"globalFPDAppDataValue","inputFPDAppData":"inputFPDAppDataValue"}}`)},
		},
		{
			description:   "FPD config, bid request and global fpd app are specified, with input app ext.data malformed",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
			globalFPD:     map[string][]byte{appKey: []byte(`malformed`)},
			expectError:   "invalid first party data ext",
		},
		{
			description:   "bid request and openrtb global fpd app are specified, no input app ext",
			bidRequestApp: &openrtb2.App{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{appContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedApp: &openrtb2.App{ID: "test2", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}}},
		},
		{
			description: "bid request with content and openrtb global fpd app are specified, no input app ext",
			bidRequestApp: &openrtb2.App{ID: "test2", Content: &openrtb2.Content{
				ID: "InputAppContentId",
				Data: []openrtb2.Data{
					{ID: "1", Name: "N1"},
					{ID: "2", Name: "N2"},
				},
				Ext: json.RawMessage(`{"contentPresent":true}`),
			}},
			openRtbGlobalFPD: map[string][]openrtb2.Data{appContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedApp: &openrtb2.App{ID: "test2", Content: &openrtb2.Content{
				ID: "InputAppContentId",
				Data: []openrtb2.Data{
					{ID: "DataId1", Name: "Name1"},
					{ID: "DataId2", Name: "Name2"},
				},
				Ext: json.RawMessage(`{"contentPresent":true}`),
			}},
		},
		{
			description:   "fpd config app, bid request and openrtb global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{appContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedApp: &openrtb2.App{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}}},
		},
		{
			description:   "fpd config app with ext, bid request and openrtb global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1","ext":{"test":1}}`)},
			bidRequestApp: &openrtb2.App{ID: "test2"},
			openRtbGlobalFPD: map[string][]openrtb2.Data{appContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedApp: &openrtb2.App{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
				Ext: json.RawMessage(`{"test":1}`)},
		},
		{
			description:   "fpd config app with ext, bid request app with ext and openrtb global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1","ext":{"test":1}}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"test":2,"key":"value"}`)},
			openRtbGlobalFPD: map[string][]openrtb2.Data{appContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedApp: &openrtb2.App{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
				Ext: json.RawMessage(`{"key":"value","test":1}`)},
		},
		{
			description:   "fpd config app with malformed ext, bid request app with ext and openrtb global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id":"test1","ext":{malformed}}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"test":2,"key":"value"}`)},
			openRtbGlobalFPD: map[string][]openrtb2.Data{appContentDataKey: {
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
			expectedApp: &openrtb2.App{ID: "test1", Content: &openrtb2.Content{Data: []openrtb2.Data{
				{ID: "DataId1", Name: "Name1"},
				{ID: "DataId2", Name: "Name2"},
			}},
				Ext: json.RawMessage(`{"key":"value","test":1}`),
			},
			expectError: "invalid first party data ext",
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resultApp, err := resolveApp(test.fpdConfig, test.bidRequestApp, test.globalFPD, test.openRtbGlobalFPD, "bidderA")

			if len(test.expectError) > 0 {
				assert.EqualError(t, err, test.expectError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedApp, resultApp, "Result app is incorrect")
			}
		})
	}
}

func TestResolveDevice(t *testing.T) {
	testCases := []struct {
		description      string
		fpdConfig        *openrtb_ext.ORTB2
		bidRequestDevice *openrtb2.Device
		expectedDevice   *openrtb2.Device
		expectError      string
	}{
		{
			description:    "FPD config and bid request device are not specified",
			expectedDevice: nil,
		},
		{
			description: "FPD config device only is specified",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"ua":"test-user-agent"}`)},
			expectedDevice: &openrtb2.Device{
				UA: "test-user-agent",
			},
		},
		{
			description:      "FPD config and bid request device are specified",
			fpdConfig:        &openrtb_ext.ORTB2{Device: json.RawMessage(`{"ua":"test-user-agent-1"}`)},
			bidRequestDevice: &openrtb2.Device{UA: "test-user-agent-2"},
			expectedDevice:   &openrtb2.Device{UA: "test-user-agent-1"},
		},
		{
			description:      "Bid request device only is specified",
			bidRequestDevice: &openrtb2.Device{UA: "test-user-agent"},
			expectedDevice:   &openrtb2.Device{UA: "test-user-agent"},
		},
		{
			description: "FPD config device with ext, bid request device with ext",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"ua":"test-user-agent-1","ext":{"test":1}}`)},
			bidRequestDevice: &openrtb2.Device{
				UA:  "test-user-agent-2",
				Ext: json.RawMessage(`{"test":2,"key":"value"}`),
			},
			expectedDevice: &openrtb2.Device{
				UA:  "test-user-agent-1",
				Ext: json.RawMessage(`{"key":"value","test":1}`),
			},
		},
		{
			description:      "Bid request device with ext only is specified",
			bidRequestDevice: &openrtb2.Device{UA: "test-user-agent", Ext: json.RawMessage(`{"customData":true}`)},
			expectedDevice:   &openrtb2.Device{UA: "test-user-agent", Ext: json.RawMessage(`{"customData":true}`)},
		},
		{
			description: "FPD config device with malformed ext",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"ua":"test-user-agent-1","ext":{malformed}}`)},
			bidRequestDevice: &openrtb2.Device{
				UA:  "test-user-agent-2",
				Ext: json.RawMessage(`{"test":2,"key":"value"}`),
			},
			expectError: "invalid first party data ext",
		},
		{
			description: "Device with negative width should fail validation",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"w":-10}`)},
			expectError: "request.device.w must be a positive number",
		},
		{
			description: "Device with negative height should fail validation",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"h":-20}`)},
			expectError: "request.device.h must be a positive number",
		},
		{
			description: "Device with negative PPI should fail validation",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"ppi":-300}`)},
			expectError: "request.device.ppi must be a positive number",
		},
		{
			description: "Device with negative geo accuracy should fail validation",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"geo":{"accuracy":-5}}`)},
			expectError: "request.device.geo.accuracy must be a positive number",
		},
		{
			description: "Merging valid device properties should pass validation",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"w":100,"h":200,"ppi":300}`)},
			bidRequestDevice: &openrtb2.Device{
				UA:  "test-user-agent",
				Geo: &openrtb2.Geo{Accuracy: 10},
			},
			expectedDevice: &openrtb2.Device{
				UA:  "test-user-agent",
				W:   100,
				H:   200,
				PPI: 300,
				Geo: &openrtb2.Geo{Accuracy: 10},
			},
		},
		{
			description: "Resulting merged device with negative values should fail validation",
			fpdConfig:   &openrtb_ext.ORTB2{Device: json.RawMessage(`{"w":-100}`)},
			bidRequestDevice: &openrtb2.Device{
				UA: "test-user-agent",
				H:  200,
			},
			expectError: "request.device.w must be a positive number",
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resultDevice, err := resolveDevice(test.fpdConfig, test.bidRequestDevice)

			if len(test.expectError) > 0 {
				assert.EqualError(t, err, test.expectError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedDevice, resultDevice, "Result device is incorrect")
			}
		})
	}
}

func TestBuildExtData(t *testing.T) {
	testCases := []struct {
		description string
		input       []byte
		expectedRes string
	}{
		{
			description: "Input object with int value",
			input:       []byte(`{"someData":123}`),
			expectedRes: `{"data":{"someData":123}}`,
		},
		{
			description: "Input object with bool value",
			input:       []byte(`{"someData":true}`),
			expectedRes: `{"data":{"someData":true}}`,
		},
		{
			description: "Input object with string value",
			input:       []byte(`{"someData":"true"}`),
			expectedRes: `{"data":{"someData":"true"}}`,
		},
		{
			description: "No input object",
			input:       []byte(`{}`),
			expectedRes: `{"data":{}}`,
		},
		{
			description: "Input object with object value",
			input:       []byte(`{"someData":{"moreFpdData":"fpddata"}}`),
			expectedRes: `{"data":{"someData":{"moreFpdData":"fpddata"}}}`,
		},
	}

	for _, test := range testCases {
		actualRes := buildExtData(test.input)
		assert.JSONEq(t, test.expectedRes, string(actualRes), "Incorrect result data")
	}
}

func loadTestFile[T any](filename string) (T, error) {
	var testFile T

	b, err := os.ReadFile(filename)
	if err != nil {
		return testFile, err
	}

	err = json.Unmarshal(b, &testFile)
	if err != nil {
		return testFile, err
	}

	return testFile, nil
}

type fpdFile struct {
	InputRequestData   json.RawMessage                                    `json:"inputRequestData,omitempty"`
	OutputRequestData  json.RawMessage                                    `json:"outputRequestData,omitempty"`
	BidderConfigFPD    map[openrtb_ext.BidderName]*openrtb_ext.ORTB2      `json:"bidderConfigFPD,omitempty"`
	BiddersFPDResolved map[openrtb_ext.BidderName]*ResolvedFirstPartyData `json:"biddersFPDResolved,omitempty"`
	GlobalFPD          map[string]json.RawMessage                         `json:"globalFPD,omitempty"`
	ValidationErrors   []*errortypes.BadInput                             `json:"validationErrors,omitempty"`
}

type fpdFileForResolveFPD struct {
	InputRequestData     json.RawMessage                                `json:"inputRequestData,omitempty"`
	OutputRequestData    map[openrtb_ext.BidderName]openrtb2.BidRequest `json:"outputRequestData,omitempty"`
	BiddersWithGlobalFPD []string                                       `json:"biddersWithGlobalFPD,omitempty"`
	BidderConfigFPD      map[openrtb_ext.BidderName]*openrtb_ext.ORTB2  `json:"bidderConfigFPD,omitempty"`
	GlobalFPD            map[string]json.RawMessage                     `json:"globalFPD,omitempty"`
	ValidationErrors     []*errortypes.BadInput                         `json:"validationErrors,omitempty"`
}

func TestValidateDevice(t *testing.T) {
	tests := []struct {
		name          string
		device        *openrtb2.Device
		expectedError error
	}{
		{
			name:          "nil device",
			device:        nil,
			expectedError: nil,
		},
		{
			name:          "valid device",
			device:        &openrtb2.Device{W: 0, H: 0, PPI: 0},
			expectedError: nil,
		},
		{
			name:          "valid device with positive values",
			device:        &openrtb2.Device{W: 300, H: 250, PPI: 326},
			expectedError: nil,
		},
		{
			name:          "negative width",
			device:        &openrtb2.Device{W: -1, H: 0, PPI: 0},
			expectedError: errors.New("request.device.w must be a positive number"),
		},
		{
			name:          "negative height",
			device:        &openrtb2.Device{W: 0, H: -1, PPI: 0},
			expectedError: errors.New("request.device.h must be a positive number"),
		},
		{
			name:          "negative PPI",
			device:        &openrtb2.Device{W: 0, H: 0, PPI: -1},
			expectedError: errors.New("request.device.ppi must be a positive number"),
		},
		{
			name:          "nil geo",
			device:        &openrtb2.Device{W: 0, H: 0, PPI: 0, Geo: nil},
			expectedError: nil,
		},
		{
			name:          "valid geo accuracy",
			device:        &openrtb2.Device{W: 0, H: 0, PPI: 0, Geo: &openrtb2.Geo{Accuracy: 0}},
			expectedError: nil,
		},
		{
			name:          "positive geo accuracy",
			device:        &openrtb2.Device{W: 0, H: 0, PPI: 0, Geo: &openrtb2.Geo{Accuracy: 10}},
			expectedError: nil,
		},
		{
			name:          "negative geo accuracy",
			device:        &openrtb2.Device{W: 0, H: 0, PPI: 0, Geo: &openrtb2.Geo{Accuracy: -1}},
			expectedError: errors.New("request.device.geo.accuracy must be a positive number"),
		},
		{
			name:          "mixed valid and invalid fields",
			device:        &openrtb2.Device{W: 300, H: -1, PPI: -2, Geo: &openrtb2.Geo{Accuracy: -3}},
			expectedError: errors.New("request.device.h must be a positive number"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDevice(tt.device)
			if tt.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedError.Error())
			}
		})
	}
}
