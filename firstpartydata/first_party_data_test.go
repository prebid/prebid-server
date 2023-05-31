package firstpartydata

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
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
						Ext: json.RawMessage(`{"data": {"somesitefpd": "sitefpdDataTest"}}`),
					},
					User: &openrtb2.User{
						ID:     "reqUserID",
						Yob:    1982,
						Gender: "M",
						Ext:    json.RawMessage(`{"data": {"someuserfpd": "userfpdDataTest"}}`),
					},
					App: &openrtb2.App{
						ID:  "appId",
						Ext: json.RawMessage(`{"data": {"someappfpd": "appfpdDataTest"}}`),
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
				"site": []byte(`{"somesitefpd": "sitefpdDataTest"}`),
				"user": []byte(`{"someuserfpd": "userfpdDataTest"}`),
				"app":  []byte(`{"someappfpd": "appfpdDataTest"}`),
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
						Ext: json.RawMessage(`{"data": {"someappfpd": "appfpdDataTest"}}`),
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
				"app":  []byte(`{"someappfpd": "appfpdDataTest"}`),
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
						Ext:    json.RawMessage(`{"data": {"someuserfpd": "userfpdDataTest"}}`),
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
				"user": []byte(`{"someuserfpd": "userfpdDataTest"}`),
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
						Ext: json.RawMessage(`{"data": {"someappfpd": true}}`),
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
				"site": []byte(`{"someappfpd": true}`),
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
			filePath := testPath + "/" + test.Name()

			fpdFile, err := loadFpdFile(filePath)
			require.NoError(t, err, "Cannot Load Test")

			givenRequestExtPrebid := &openrtb_ext.ExtRequestPrebid{}
			err = json.Unmarshal(fpdFile.InputRequestData, givenRequestExtPrebid)
			require.NoError(t, err, "Cannot Load Test Conditions")

			testRequest := &openrtb_ext.RequestExt{}
			testRequest.SetPrebid(givenRequestExtPrebid)

			// run test
			results, err := ExtractBidderConfigFPD(testRequest)

			// assert errors
			if len(fpdFile.ValidationErrors) > 0 {
				require.EqualError(t, err, fpdFile.ValidationErrors[0].Message, "Expected Error Not Received")
			} else {
				require.NoError(t, err, "Error Not Expected")
				assert.Nil(t, testRequest.GetPrebid().BidderConfigs, "Bidder specific FPD config should be removed from request")
			}

			// assert fpd (with normalization for nicer looking tests)
			for bidderName, expectedFPD := range fpdFile.BidderConfigFPD {
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
			filePath := testPath + "/" + test.Name()

			fpdFile, err := loadFpdFile(filePath)
			require.NoError(t, err, "Cannot Load Test")

			request := &openrtb2.BidRequest{}
			err = json.Unmarshal(fpdFile.InputRequestData, &request)
			require.NoError(t, err, "Cannot Load Request")

			originalRequest := &openrtb2.BidRequest{}
			err = json.Unmarshal(fpdFile.InputRequestData, &originalRequest)
			require.NoError(t, err, "Cannot Load Request")

			outputReq := &openrtb2.BidRequest{}
			err = json.Unmarshal(fpdFile.OutputRequestData, &outputReq)
			require.NoError(t, err, "Cannot Load Output Request")

			reqExtFPD := make(map[string][]byte)
			reqExtFPD["site"] = fpdFile.GlobalFPD["site"]
			reqExtFPD["app"] = fpdFile.GlobalFPD["app"]
			reqExtFPD["user"] = fpdFile.GlobalFPD["user"]

			reqFPD := make(map[string][]openrtb2.Data, 3)

			reqFPDSiteContentData := fpdFile.GlobalFPD[siteContentDataKey]
			if len(reqFPDSiteContentData) > 0 {
				var siteConData []openrtb2.Data
				err = json.Unmarshal(reqFPDSiteContentData, &siteConData)
				if err != nil {
					t.Errorf("Unable to unmarshal site.content.data:")
				}
				reqFPD[siteContentDataKey] = siteConData
			}

			reqFPDAppContentData := fpdFile.GlobalFPD[appContentDataKey]
			if len(reqFPDAppContentData) > 0 {
				var appConData []openrtb2.Data
				err = json.Unmarshal(reqFPDAppContentData, &appConData)
				if err != nil {
					t.Errorf("Unable to unmarshal app.content.data: ")
				}
				reqFPD[appContentDataKey] = appConData
			}

			reqFPDUserData := fpdFile.GlobalFPD[userDataKey]
			if len(reqFPDUserData) > 0 {
				var userData []openrtb2.Data
				err = json.Unmarshal(reqFPDUserData, &userData)
				if err != nil {
					t.Errorf("Unable to unmarshal app.content.data: ")
				}
				reqFPD[userDataKey] = userData
			}
			if fpdFile.BidderConfigFPD == nil {
				fpdFile.BidderConfigFPD = make(map[openrtb_ext.BidderName]*openrtb_ext.ORTB2)
				fpdFile.BidderConfigFPD["appnexus"] = &openrtb_ext.ORTB2{}
			}

			// run test
			resultFPD, errL := ResolveFPD(request, fpdFile.BidderConfigFPD, reqExtFPD, reqFPD, []string{"appnexus"})

			if len(errL) == 0 {
				assert.Equal(t, request, originalRequest, "Original request should not be modified")

				bidderFPD := resultFPD["appnexus"]

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
			} else {
				assert.ElementsMatch(t, errL, fpdFile.ValidationErrors, "Incorrect first party data warning message")
			}
		})
	}
}

func TestExtractFPDForBidders(t *testing.T) {
	if specFiles, err := os.ReadDir("./tests/extractfpdforbidders"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./tests/extractfpdforbidders/" + specFile.Name()

			fpdFile, err := loadFpdFile(fileName)

			if err != nil {
				t.Errorf("Unable to load file: %s", fileName)
			}

			var expectedRequest openrtb2.BidRequest
			err = json.Unmarshal(fpdFile.OutputRequestData, &expectedRequest)
			if err != nil {
				t.Errorf("Unable to unmarshal input request: %s", fileName)
			}

			resultRequest := &openrtb_ext.RequestWrapper{}
			resultRequest.BidRequest = &openrtb2.BidRequest{}
			err = json.Unmarshal(fpdFile.InputRequestData, resultRequest.BidRequest)
			assert.NoError(t, err, "Error should be nil")

			resultFPD, errL := ExtractFPDForBidders(resultRequest)

			if len(fpdFile.ValidationErrors) > 0 {
				assert.Equal(t, len(fpdFile.ValidationErrors), len(errL), "Incorrect number of errors was returned")
				assert.ElementsMatch(t, errL, fpdFile.ValidationErrors, "Incorrect errors were returned")
				//in case or error no further assertions needed
				continue
			}
			assert.Empty(t, errL, "Error should be empty")
			assert.Equal(t, len(resultFPD), len(fpdFile.BiddersFPDResolved))

			for bidderName, expectedValue := range fpdFile.BiddersFPDResolved {
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

func loadFpdFile(filename string) (fpdFile, error) {
	var fileData fpdFile
	fileContents, err := os.ReadFile(filename)
	if err != nil {
		return fileData, err
	}
	err = json.Unmarshal(fileContents, &fileData)
	if err != nil {
		return fileData, err
	}

	return fileData, nil
}

type fpdFile struct {
	InputRequestData   json.RawMessage                                    `json:"inputRequestData,omitempty"`
	OutputRequestData  json.RawMessage                                    `json:"outputRequestData,omitempty"`
	BidderConfigFPD    map[openrtb_ext.BidderName]*openrtb_ext.ORTB2      `json:"bidderConfigFPD,omitempty"`
	BiddersFPDResolved map[openrtb_ext.BidderName]*ResolvedFirstPartyData `json:"biddersFPDResolved,omitempty"`
	GlobalFPD          map[string]json.RawMessage                         `json:"globalFPD,omitempty"`
	ValidationErrors   []*errortypes.BadInput                             `json:"validationErrors,omitempty"`
}

func TestResolveUser(t *testing.T) {
	testCases := []struct {
		description      string
		fpdConfig        *openrtb_ext.ORTB2
		bidRequestUser   *openrtb2.User
		globalFPD        map[string][]byte
		openRtbGlobalFPD map[string][]openrtb2.Data
		expectedUser     *openrtb2.User
		expectedError    string
	}{
		{
			description:  "FPD config and bid request user are not specified",
			expectedUser: nil,
		},
		{
			description:  "FPD config user only is specified",
			fpdConfig:    &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test"}`)},
			expectedUser: &openrtb2.User{ID: "test"},
		},
		{
			description:    "FPD config and bid request user are specified",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2"},
			expectedUser:   &openrtb2.User{ID: "test1"},
		},
		{
			description:    "FPD config, bid request and global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2"},
			globalFPD:      map[string][]byte{userKey: []byte(`{"globalFPDUserData": "globalFPDUserDataValue"}`)},
			expectedUser:   &openrtb2.User{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDUserData":"globalFPDUserDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request user with ext and global fpd user are specified, no input user ext",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"test":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
			globalFPD:      map[string][]byte{userKey: []byte(`{"globalFPDUserData": "globalFPDUserDataValue"}`)},
			expectedUser:   &openrtb2.User{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDUserData":"globalFPDUserDataValue"},"test":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd user are specified, with input user ext.data",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
			globalFPD:      map[string][]byte{userKey: []byte(`{"globalFPDUserData": "globalFPDUserDataValue"}`)},
			expectedUser:   &openrtb2.User{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDUserData":"globalFPDUserDataValue","inputFPDUserData":"inputFPDUserDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd user are specified, with input user ext.data malformed",
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1"}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDUserData":"inputFPDUserDataValue"}}`)},
			globalFPD:      map[string][]byte{userKey: []byte(`malformed`)},
			expectedError:  "Invalid JSON Patch",
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
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1"}`)},
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
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1", "ext":{"test":1}}`)},
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
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1", "ext":{"test":1}}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"test":2, "key": "value"}`)},
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
			fpdConfig:      &openrtb_ext.ORTB2{User: json.RawMessage(`{"id": "test1", "ext":{malformed}}`)},
			bidRequestUser: &openrtb2.User{ID: "test2", Ext: json.RawMessage(`{"test":2, "key": "value"}`)},
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
			expectedError: "invalid character 'm' looking for beginning of object key string",
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resultUser, err := resolveUser(test.fpdConfig, test.bidRequestUser, test.globalFPD, test.openRtbGlobalFPD, "bidderA")

			if test.expectedError == "" {
				assert.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedUser, resultUser, "Result user is incorrect")
			} else {
				assert.EqualError(t, err, test.expectedError, "expected error incorrect")
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
		expectedError    string
	}{
		{
			description:  "FPD config and bid request site are not specified",
			expectedSite: nil,
		},
		{
			description:   "FPD config site only is specified",
			fpdConfig:     &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test"}`)},
			expectedError: "incorrect First Party Data for bidder bidderA: Site object is not defined in request, but defined in FPD config",
		},
		{
			description:    "FPD config and bid request site are specified",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2"},
			expectedSite:   &openrtb2.Site{ID: "test1"},
		},
		{
			description:    "FPD config, bid request and global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2"},
			globalFPD:      map[string][]byte{siteKey: []byte(`{"globalFPDSiteData": "globalFPDSiteDataValue"}`)},
			expectedSite:   &openrtb2.Site{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDSiteData":"globalFPDSiteDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request site with ext and global fpd site are specified, no input site ext",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"test":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
			globalFPD:      map[string][]byte{siteKey: []byte(`{"globalFPDSiteData": "globalFPDSiteDataValue"}`)},
			expectedSite:   &openrtb2.Site{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDSiteData":"globalFPDSiteDataValue"},"test":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd site are specified, with input site ext.data",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
			globalFPD:      map[string][]byte{siteKey: []byte(`{"globalFPDSiteData": "globalFPDSiteDataValue"}`)},
			expectedSite:   &openrtb2.Site{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDSiteData":"globalFPDSiteDataValue","inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
		},
		{
			description:    "FPD config, bid request and global fpd site are specified, with input site ext.data malformed",
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1"}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDSiteData":"inputFPDSiteDataValue"}}`)},
			globalFPD:      map[string][]byte{siteKey: []byte(`malformed`)},
			expectedError:  "Invalid JSON Patch",
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
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1"}`)},
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
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1", "ext":{"test":1}}`)},
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
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1", "ext":{"test":1}}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"test":2, "key": "value"}`)},
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
			fpdConfig:      &openrtb_ext.ORTB2{Site: json.RawMessage(`{"id": "test1", "ext":{malformed}}`)},
			bidRequestSite: &openrtb2.Site{ID: "test2", Ext: json.RawMessage(`{"test":2, "key": "value"}`)},
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
			expectedError: "invalid character 'm' looking for beginning of object key string",
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resultSite, err := resolveSite(test.fpdConfig, test.bidRequestSite, test.globalFPD, test.openRtbGlobalFPD, "bidderA")

			if test.expectedError == "" {
				assert.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedSite, resultSite, "Result site is incorrect")
			} else {
				assert.EqualError(t, err, test.expectedError, "expected error incorrect")
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
		expectedError    string
	}{
		{
			description: "FPD config and bid request app are not specified",
			expectedApp: nil,
		},
		{
			description:   "FPD config app only is specified",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test"}`)},
			expectedError: "incorrect First Party Data for bidder bidderA: App object is not defined in request, but defined in FPD config",
		},
		{
			description:   "FPD config and bid request app are specified",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2"},
			expectedApp:   &openrtb2.App{ID: "test1"},
		},
		{
			description:   "FPD config, bid request and global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2"},
			globalFPD:     map[string][]byte{appKey: []byte(`{"globalFPDAppData": "globalFPDAppDataValue"}`)},
			expectedApp:   &openrtb2.App{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDAppData":"globalFPDAppDataValue"}}`)},
		},
		{
			description:   "FPD config, bid request app with ext and global fpd app are specified, no input app ext",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"test":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
			globalFPD:     map[string][]byte{appKey: []byte(`{"globalFPDAppData": "globalFPDAppDataValue"}`)},
			expectedApp:   &openrtb2.App{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDAppData":"globalFPDAppDataValue"},"test":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
		},
		{
			description:   "FPD config, bid request and global fpd app are specified, with input app ext.data",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
			globalFPD:     map[string][]byte{appKey: []byte(`{"globalFPDAppData": "globalFPDAppDataValue"}`)},
			expectedApp:   &openrtb2.App{ID: "test1", Ext: json.RawMessage(`{"data":{"globalFPDAppData":"globalFPDAppDataValue","inputFPDAppData":"inputFPDAppDataValue"}}`)},
		},
		{
			description:   "FPD config, bid request and global fpd app are specified, with input app ext.data malformed",
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1"}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"data":{"inputFPDAppData":"inputFPDAppDataValue"}}`)},
			globalFPD:     map[string][]byte{appKey: []byte(`malformed`)},
			expectedError: "Invalid JSON Patch",
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
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1"}`)},
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
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1", "ext":{"test":1}}`)},
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
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1", "ext":{"test":1}}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"test":2, "key": "value"}`)},
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
			fpdConfig:     &openrtb_ext.ORTB2{App: json.RawMessage(`{"id": "test1", "ext":{malformed}}`)},
			bidRequestApp: &openrtb2.App{ID: "test2", Ext: json.RawMessage(`{"test":2, "key": "value"}`)},
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
			expectedError: "invalid character 'm' looking for beginning of object key string",
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resultApp, err := resolveApp(test.fpdConfig, test.bidRequestApp, test.globalFPD, test.openRtbGlobalFPD, "bidderA")

			if test.expectedError == "" {
				assert.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedApp, resultApp, "Result app is incorrect")
			} else {
				assert.EqualError(t, err, test.expectedError, "expected error incorrect")
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
			input:       []byte(`{"someData": 123}`),
			expectedRes: `{"data": {"someData": 123}}`,
		},
		{
			description: "Input object with bool value",
			input:       []byte(`{"someData": true}`),
			expectedRes: `{"data": {"someData": true}}`,
		},
		{
			description: "Input object with string value",
			input:       []byte(`{"someData": "true"}`),
			expectedRes: `{"data": {"someData": "true"}}`,
		},
		{
			description: "No input object",
			input:       []byte(`{}`),
			expectedRes: `{"data": {}}`,
		},
		{
			description: "Input object with object value",
			input:       []byte(`{"someData": {"moreFpdData": "fpddata"}}`),
			expectedRes: `{"data": {"someData": {"moreFpdData": "fpddata"}}}`,
		},
	}

	for _, test := range testCases {
		actualRes := buildExtData(test.input)
		assert.JSONEq(t, test.expectedRes, string(actualRes), "Incorrect result data")
	}
}

func TestMergeUser(t *testing.T) {
	testCases := []struct {
		name         string
		givenUser    openrtb2.User
		givenFPD     json.RawMessage
		expectedUser openrtb2.User
		expectedErr  string
	}{
		{
			name:         "empty",
			givenUser:    openrtb2.User{},
			givenFPD:     []byte(`{}`),
			expectedUser: openrtb2.User{},
		},
		{
			name:         "toplevel",
			givenUser:    openrtb2.User{ID: "1"},
			givenFPD:     []byte(`{"id":"2"}`),
			expectedUser: openrtb2.User{ID: "2"},
		},
		{
			name:         "toplevel-ext",
			givenUser:    openrtb2.User{Ext: []byte(`{"a":1,"b":2}`)},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}}`),
			expectedUser: openrtb2.User{Ext: []byte(`{"a":1,"b":100,"c":3}`)},
		},
		{
			name:        "toplevel-ext-err",
			givenUser:   openrtb2.User{ID: "1", Ext: []byte(`malformed`)},
			givenFPD:    []byte(`{"id":"2"}`),
			expectedErr: "invalid request ext",
		},
		{
			name:         "nested-geo",
			givenUser:    openrtb2.User{Geo: &openrtb2.Geo{Lat: 1}},
			givenFPD:     []byte(`{"geo":{"lat": 2}}`),
			expectedUser: openrtb2.User{Geo: &openrtb2.Geo{Lat: 2}},
		},
		{
			name:         "nested-geo-ext",
			givenUser:    openrtb2.User{Geo: &openrtb2.Geo{Ext: []byte(`{"a":1,"b":2}`)}},
			givenFPD:     []byte(`{"geo":{"ext":{"b":100,"c":3}}}`),
			expectedUser: openrtb2.User{Geo: &openrtb2.Geo{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:         "toplevel-ext-and-nested-geo-ext",
			givenUser:    openrtb2.User{Ext: []byte(`{"a":1,"b":2}`), Geo: &openrtb2.Geo{Ext: []byte(`{"a":10,"b":20}`)}},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}, "geo":{"ext":{"b":100,"c":3}}}`),
			expectedUser: openrtb2.User{Ext: []byte(`{"a":1,"b":100,"c":3}`), Geo: &openrtb2.Geo{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:        "nested-geo-ext-err",
			givenUser:   openrtb2.User{Geo: &openrtb2.Geo{Ext: []byte(`malformed`)}},
			givenFPD:    []byte(`{"geo":{"ext":{"b":100,"c":3}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "fpd-err",
			givenUser:   openrtb2.User{ID: "1", Ext: []byte(`{"a":1}`)},
			givenFPD:    []byte(`malformed`),
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := mergeUser(&test.givenUser, test.givenFPD)

			if test.expectedErr == "" {
				assert.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedUser, test.givenUser, "result user is incorrect")
			} else {
				assert.EqualError(t, err, test.expectedErr, "expected error incorrect")
			}
		})
	}
}

func TestMergeApp(t *testing.T) {
	testCases := []struct {
		name        string
		givenApp    openrtb2.App
		givenFPD    json.RawMessage
		expectedApp openrtb2.App
		expectedErr string
	}{
		{
			name:        "empty",
			givenApp:    openrtb2.App{},
			givenFPD:    []byte(`{}`),
			expectedApp: openrtb2.App{},
		},
		{
			name:        "toplevel",
			givenApp:    openrtb2.App{ID: "1"},
			givenFPD:    []byte(`{"id":"2"}`),
			expectedApp: openrtb2.App{ID: "2"},
		},
		{
			name:        "toplevel-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`)},
			givenFPD:    []byte(`{"ext":{"b":100,"c":3}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`)},
		},
		{
			name:        "toplevel-ext-err",
			givenApp:    openrtb2.App{ID: "1", Ext: []byte(`malformed`)},
			givenFPD:    []byte(`{"id":"2"}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-publisher",
			givenApp:    openrtb2.App{Publisher: &openrtb2.Publisher{Name: "pub1"}},
			givenFPD:    []byte(`{"publisher":{"name": "pub2"}}`),
			expectedApp: openrtb2.App{Publisher: &openrtb2.Publisher{Name: "pub2"}},
		},
		{
			name:        "nested-content",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1"}},
			givenFPD:    []byte(`{"content":{"title": "content2"}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2"}},
		},
		{
			name:        "nested-content-producer",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1", Producer: &openrtb2.Producer{Name: "producer1"}}},
			givenFPD:    []byte(`{"content":{"title": "content2", "producer":{"name":"producer2"}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2", Producer: &openrtb2.Producer{Name: "producer2"}}},
		},
		{
			name:        "nested-content-network",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1", Network: &openrtb2.Network{Name: "network1"}}},
			givenFPD:    []byte(`{"content":{"title": "content2", "network":{"name":"network2"}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2", Network: &openrtb2.Network{Name: "network2"}}},
		},
		{
			name:        "nested-content-channel",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1", Channel: &openrtb2.Channel{Name: "channel1"}}},
			givenFPD:    []byte(`{"content":{"title": "content2", "channel":{"name":"channel2"}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2", Channel: &openrtb2.Channel{Name: "channel2"}}},
		},
		{
			name:        "nested-publisher-ext",
			givenApp:    openrtb2.App{Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":1,"b":2}`)}},
			givenFPD:    []byte(`{"publisher":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:        "nested-content-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Ext: []byte(`{"a":1,"b":2}`)}},
			givenFPD:    []byte(`{"content":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:        "nested-content-producer-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenFPD:    []byte(`{"content":{"producer":{"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:        "nested-content-network-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenFPD:    []byte(`{"content":{"network":{"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:        "nested-content-channel-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenFPD:    []byte(`{"content":{"channel":{"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:        "toplevel-ext-and-nested-publisher-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":10,"b":20}`)}},
			givenFPD:    []byte(`{"ext":{"b":100,"c":3}, "publisher":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:        "toplevel-ext-and-nested-content-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Ext: []byte(`{"a":10,"b":20}`)}},
			givenFPD:    []byte(`{"ext":{"b":100,"c":3}, "content":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:        "toplevel-ext-and-nested-content-producer-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenFPD:    []byte(`{"ext":{"b":100,"c":3}, "content":{"producer": {"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:        "toplevel-ext-and-nested-content-network-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenFPD:    []byte(`{"ext":{"b":100,"c":3}, "content":{"network": {"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:        "toplevel-ext-and-nested-content-channel-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenFPD:    []byte(`{"ext":{"b":100,"c":3}, "content":{"channel": {"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:        "nested-publisher-ext-err",
			givenApp:    openrtb2.App{Publisher: &openrtb2.Publisher{Ext: []byte(`malformed`)}},
			givenFPD:    []byte(`{"publisher":{"ext":{"b":100,"c":3}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Ext: []byte(`malformed`)}},
			givenFPD:    []byte(`{"content":{"ext":{"b":100,"c":3}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-producer-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`malformed`)}}},
			givenFPD:    []byte(`{"content":{"producer": {"ext":{"b":100,"c":3}}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-network-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`malformed`)}}},
			givenFPD:    []byte(`{"content":{"network": {"ext":{"b":100,"c":3}}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-channel-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`malformed`)}}},
			givenFPD:    []byte(`{"content":{"channelx": {"ext":{"b":100,"c":3}}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "fpd-err",
			givenApp:    openrtb2.App{ID: "1", Ext: []byte(`{"a":1}`)},
			givenFPD:    []byte(`malformed`),
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := mergeApp(&test.givenApp, test.givenFPD)

			if test.expectedErr == "" {
				assert.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedApp, test.givenApp, " result app is incorrect")
			} else {
				assert.EqualError(t, err, test.expectedErr, "expected error incorrect")
			}
		})
	}
}

func TestMergeSite(t *testing.T) {
	testCases := []struct {
		name         string
		givenSite    openrtb2.Site
		givenFPD     json.RawMessage
		expectedSite openrtb2.Site
		expectedErr  string
	}{
		{
			name:        "empty",
			givenSite:   openrtb2.Site{},
			givenFPD:    []byte(`{}`),
			expectedErr: "incorrect First Party Data for bidder BidderA: Site object cannot set empty page if req.site.id is empty",
		},
		{
			name:         "toplevel",
			givenSite:    openrtb2.Site{ID: "1"},
			givenFPD:     []byte(`{"id":"2"}`),
			expectedSite: openrtb2.Site{ID: "2"},
		},
		{
			name:         "toplevel-ext",
			givenSite:    openrtb2.Site{Page: "test.com/page", Ext: []byte(`{"a":1,"b":2}`)},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}}`),
			expectedSite: openrtb2.Site{Page: "test.com/page", Ext: []byte(`{"a":1,"b":100,"c":3}`)},
		},
		{
			name:        "toplevel-ext-err",
			givenSite:   openrtb2.Site{ID: "1", Ext: []byte(`malformed`)},
			givenFPD:    []byte(`{"id":"2"}`),
			expectedErr: "invalid request ext",
		},
		{
			name:         "nested-publisher",
			givenSite:    openrtb2.Site{Page: "test.com/page", Publisher: &openrtb2.Publisher{Name: "pub1"}},
			givenFPD:     []byte(`{"publisher":{"name": "pub2"}}`),
			expectedSite: openrtb2.Site{Page: "test.com/page", Publisher: &openrtb2.Publisher{Name: "pub2"}},
		},
		{
			name:         "nested-content",
			givenSite:    openrtb2.Site{Page: "test.com/page", Content: &openrtb2.Content{Title: "content1"}},
			givenFPD:     []byte(`{"content":{"title": "content2"}}`),
			expectedSite: openrtb2.Site{Page: "test.com/page", Content: &openrtb2.Content{Title: "content2"}},
		},
		{
			name:         "nested-content-producer",
			givenSite:    openrtb2.Site{ID: "1", Content: &openrtb2.Content{Title: "content1", Producer: &openrtb2.Producer{Name: "producer1"}}},
			givenFPD:     []byte(`{"content":{"title": "content2", "producer":{"name":"producer2"}}}`),
			expectedSite: openrtb2.Site{ID: "1", Content: &openrtb2.Content{Title: "content2", Producer: &openrtb2.Producer{Name: "producer2"}}},
		},
		{
			name:         "nested-content-network",
			givenSite:    openrtb2.Site{ID: "1", Content: &openrtb2.Content{Title: "content1", Network: &openrtb2.Network{Name: "network1"}}},
			givenFPD:     []byte(`{"content":{"title": "content2", "network":{"name":"network2"}}}`),
			expectedSite: openrtb2.Site{ID: "1", Content: &openrtb2.Content{Title: "content2", Network: &openrtb2.Network{Name: "network2"}}},
		},
		{
			name:         "nested-content-channel",
			givenSite:    openrtb2.Site{ID: "1", Content: &openrtb2.Content{Title: "content1", Channel: &openrtb2.Channel{Name: "channel1"}}},
			givenFPD:     []byte(`{"content":{"title": "content2", "channel":{"name":"channel2"}}}`),
			expectedSite: openrtb2.Site{ID: "1", Content: &openrtb2.Content{Title: "content2", Channel: &openrtb2.Channel{Name: "channel2"}}},
		},
		{
			name:         "nested-publisher-ext",
			givenSite:    openrtb2.Site{ID: "1", Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":1,"b":2}`)}},
			givenFPD:     []byte(`{"publisher":{"ext":{"b":100,"c":3}}}`),
			expectedSite: openrtb2.Site{ID: "1", Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:         "nested-content-ext",
			givenSite:    openrtb2.Site{ID: "1", Content: &openrtb2.Content{Ext: []byte(`{"a":1,"b":2}`)}},
			givenFPD:     []byte(`{"content":{"ext":{"b":100,"c":3}}}`),
			expectedSite: openrtb2.Site{ID: "1", Content: &openrtb2.Content{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:         "nested-content-producer-ext",
			givenSite:    openrtb2.Site{ID: "1", Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenFPD:     []byte(`{"content":{"producer":{"ext":{"b":100,"c":3}}}}`),
			expectedSite: openrtb2.Site{ID: "1", Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:         "nested-content-network-ext",
			givenSite:    openrtb2.Site{ID: "1", Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenFPD:     []byte(`{"content":{"network":{"ext":{"b":100,"c":3}}}}`),
			expectedSite: openrtb2.Site{ID: "1", Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:         "nested-content-channel-ext",
			givenSite:    openrtb2.Site{ID: "1", Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenFPD:     []byte(`{"content":{"channel":{"ext":{"b":100,"c":3}}}}`),
			expectedSite: openrtb2.Site{ID: "1", Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:         "toplevel-ext-and-nested-publisher-ext",
			givenSite:    openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":2}`), Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":10,"b":20}`)}},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}, "publisher":{"ext":{"b":100,"c":3}}}`),
			expectedSite: openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":100,"c":3}`), Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:         "toplevel-ext-and-nested-content-ext",
			givenSite:    openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Ext: []byte(`{"a":10,"b":20}`)}},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}, "content":{"ext":{"b":100,"c":3}}}`),
			expectedSite: openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:         "toplevel-ext-and-nested-content-producer-ext",
			givenSite:    openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}, "content":{"producer": {"ext":{"b":100,"c":3}}}}`),
			expectedSite: openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:         "toplevel-ext-and-nested-content-network-ext",
			givenSite:    openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}, "content":{"network": {"ext":{"b":100,"c":3}}}}`),
			expectedSite: openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:         "toplevel-ext-and-nested-content-channel-ext",
			givenSite:    openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenFPD:     []byte(`{"ext":{"b":100,"c":3}, "content":{"channel": {"ext":{"b":100,"c":3}}}}`),
			expectedSite: openrtb2.Site{ID: "1", Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:        "nested-publisher-ext-err",
			givenSite:   openrtb2.Site{ID: "1", Publisher: &openrtb2.Publisher{Ext: []byte(`malformed`)}},
			givenFPD:    []byte(`{"publisher":{"ext":{"b":100,"c":3}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-ext-err",
			givenSite:   openrtb2.Site{ID: "1", Content: &openrtb2.Content{Ext: []byte(`malformed`)}},
			givenFPD:    []byte(`{"content":{"ext":{"b":100,"c":3}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-producer-ext-err",
			givenSite:   openrtb2.Site{ID: "1", Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`malformed`)}}},
			givenFPD:    []byte(`{"content":{"producer": {"ext":{"b":100,"c":3}}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-network-ext-err",
			givenSite:   openrtb2.Site{ID: "1", Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`malformed`)}}},
			givenFPD:    []byte(`{"content":{"network": {"ext":{"b":100,"c":3}}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "nested-content-channel-ext-err",
			givenSite:   openrtb2.Site{ID: "1", Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`malformed`)}}},
			givenFPD:    []byte(`{"content":{"channelx": {"ext":{"b":100,"c":3}}}}`),
			expectedErr: "invalid request ext",
		},
		{
			name:        "fpd-err",
			givenSite:   openrtb2.Site{ID: "1", Ext: []byte(`{"a":1}`)},
			givenFPD:    []byte(`malformed`),
			expectedErr: "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := mergeSite(&test.givenSite, test.givenFPD, "BidderA")

			if test.expectedErr == "" {
				assert.NoError(t, err, "unexpected error returned")
				assert.Equal(t, test.expectedSite, test.givenSite, " result Site is incorrect")
			} else {
				assert.EqualError(t, err, test.expectedErr, "expected error incorrect")
			}
		})
	}
}

// TestMergeObjectStructure detects when new nested objects are added to First Party Data supported
// fields, as these will invalidate the mergeSite, mergeApp, and mergeUser methods. If this test fails,
// fix the merge methods to add support and update this test to set a new baseline.
func TestMergeObjectStructure(t *testing.T) {
	testCases := []struct {
		name         string
		kind         any
		knownStructs []string
	}{
		{
			name: "Site",
			kind: openrtb2.Site{},
			knownStructs: []string{
				"Publisher",
				"Content",
				"Content.Producer",
				"Content.Network",
				"Content.Channel",
			},
		},
		{
			name: "App",
			kind: openrtb2.App{},
			knownStructs: []string{
				"Publisher",
				"Content",
				"Content.Producer",
				"Content.Network",
				"Content.Channel",
			},
		},
		{
			name: "User",
			kind: openrtb2.User{},
			knownStructs: []string{
				"Geo",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			nestedStructs := []string{}

			var discover func(parent string, t reflect.Type)
			discover = func(parent string, t reflect.Type) {
				fields := reflect.VisibleFields(t)
				for _, field := range fields {
					if field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct {
						nestedStructs = append(nestedStructs, parent+field.Name)
						discover(parent+field.Name+".", field.Type.Elem())
					}
				}
			}
			discover("", reflect.TypeOf(test.kind))

			assert.ElementsMatch(t, test.knownStructs, nestedStructs)
		})
	}
}

// user memory protect test
func TestMergeUserMemoryProtection(t *testing.T) {
	inputGeo := &openrtb2.Geo{
		Ext: json.RawMessage(`{"a":1,"b":2}`),
	}
	input := openrtb2.User{
		ID:  "1",
		Geo: inputGeo,
	}

	err := mergeUser(&input, userFPD)
	assert.NoError(t, err)

	// Input user object is expected to be a copy. Changes are ok.
	assert.Equal(t, "2", input.ID, "user-id-copied")

	// Nested objects must be copied before changes.
	assert.JSONEq(t, `{"a":1,"b":2}`, string(inputGeo.Ext), "geo-input")
	assert.JSONEq(t, `{"a":1,"b":100,"c":3}`, string(input.Geo.Ext), "geo-copied")
}

// app memory protect test
func TestMergeAppMemoryProtection(t *testing.T) {
	inputPublisher := &openrtb2.Publisher{
		ID:  "InPubId",
		Ext: json.RawMessage(`{"a": "inputPubExt", "b": 1}`),
	}
	inputContent := &openrtb2.Content{
		ID:  "InContentId",
		Ext: json.RawMessage(`{"a": "inputContentExt", "b": 1}`),
		Producer: &openrtb2.Producer{
			ID:  "InProducerId",
			Ext: json.RawMessage(`{"a": "inputProducerExt", "b": 1}`),
		},
		Network: &openrtb2.Network{
			ID:  "InNetworkId",
			Ext: json.RawMessage(`{"a": "inputNetworkExt", "b": 1}`),
		},
		Channel: &openrtb2.Channel{
			ID:  "InChannelId",
			Ext: json.RawMessage(`{"a": "inputChannelExt", "b": 1}`),
		},
	}
	input := openrtb2.App{
		ID:        "InAppID",
		Publisher: inputPublisher,
		Content:   inputContent,
		Ext:       json.RawMessage(`{"a": "inputAppExt", "b": 1}`),
	}

	err := mergeApp(&input, fpdWithPublisherAndContent)
	assert.NoError(t, err)

	// Input app object is expected to be a copy. Changes are ok.
	assert.Equal(t, "FPDID", input.ID, "app-id-copied")
	assert.JSONEq(t, `{"a": "FPDExt", "b": 2}`, string(input.Ext), "app-ext-copied")

	// Nested objects must be copied before changes.
	assert.Equal(t, "InPubId", inputPublisher.ID, "app-pub-id-input")
	assert.Equal(t, "FPDPubId", input.Publisher.ID, "app-pub-id-copied")
	assert.JSONEq(t, `{"a": "inputPubExt", "b": 1}`, string(inputPublisher.Ext), "app-pub-ext-input")
	assert.JSONEq(t, `{"a": "FPDPubExt", "b": 2}`, string(input.Publisher.Ext), "app-pub-ext-copied")

	assert.Equal(t, "InContentId", inputContent.ID, "app-content-id-input")
	assert.Equal(t, "FPDContentId", input.Content.ID, "app-content-id-copied")
	assert.JSONEq(t, `{"a": "inputContentExt", "b": 1}`, string(inputContent.Ext), "app-content-ext-input")
	assert.JSONEq(t, `{"a": "FPDContentExt", "b": 2}`, string(input.Content.Ext), "app-content-ext-copied")

	assert.Equal(t, "InProducerId", inputContent.Producer.ID, "app-content-producer-id-input")
	assert.Equal(t, "FPDProducerId", input.Content.Producer.ID, "app-content-producer-id-copied")
	assert.JSONEq(t, `{"a": "inputProducerExt", "b": 1}`, string(inputContent.Producer.Ext), "app-content-producer-ext-input")
	assert.JSONEq(t, `{"a": "FPDProducerExt", "b": 2}`, string(input.Content.Producer.Ext), "app-content-producer-ext-copied")

	assert.Equal(t, "InNetworkId", inputContent.Network.ID, "app-content-network-id-input")
	assert.Equal(t, "FPDNetworkId", input.Content.Network.ID, "app-content-network-id-copied")
	assert.JSONEq(t, `{"a": "inputNetworkExt", "b": 1}`, string(inputContent.Network.Ext), "app-content-network-ext-input")
	assert.JSONEq(t, `{"a": "FPDNetworkExt", "b": 2}`, string(input.Content.Network.Ext), "app-content-network-ext-copied")

	assert.Equal(t, "InChannelId", inputContent.Channel.ID, "app-content-channel-id-input")
	assert.Equal(t, "FPDChannelId", input.Content.Channel.ID, "app-content-channel-id-copied")
	assert.JSONEq(t, `{"a": "inputChannelExt", "b": 1}`, string(inputContent.Channel.Ext), "app-content-channel-ext-input")
	assert.JSONEq(t, `{"a": "FPDChannelExt", "b": 2}`, string(input.Content.Channel.Ext), "app-content-channel-ext-copied")
}

// site memory protect test
func TestMergeSiteMemoryProtection(t *testing.T) {
	inputPublisher := &openrtb2.Publisher{
		ID:  "InPubId",
		Ext: json.RawMessage(`{"a": "inputPubExt", "b": 1}`),
	}
	inputContent := &openrtb2.Content{
		ID:  "InContentId",
		Ext: json.RawMessage(`{"a": "inputContentExt", "b": 1}`),
		Producer: &openrtb2.Producer{
			ID:  "InProducerId",
			Ext: json.RawMessage(`{"a": "inputProducerExt", "b": 1}`),
		},
		Network: &openrtb2.Network{
			ID:  "InNetworkId",
			Ext: json.RawMessage(`{"a": "inputNetworkExt", "b": 1}`),
		},
		Channel: &openrtb2.Channel{
			ID:  "InChannelId",
			Ext: json.RawMessage(`{"a": "inputChannelExt", "b": 1}`),
		},
	}
	input := openrtb2.Site{
		ID:        "InSiteID",
		Publisher: inputPublisher,
		Content:   inputContent,
		Ext:       json.RawMessage(`{"a": "inputSiteExt", "b": 1}`),
	}

	err := mergeSite(&input, fpdWithPublisherAndContent, "BidderA")
	assert.NoError(t, err)

	// Input app object is expected to be a copy. Changes are ok.
	assert.Equal(t, "FPDID", input.ID, "site-id-copied")
	assert.JSONEq(t, `{"a": "FPDExt", "b": 2}`, string(input.Ext), "site-ext-copied")

	// Nested objects must be copied before changes.
	assert.Equal(t, "InPubId", inputPublisher.ID, "site-pub-id-input")
	assert.Equal(t, "FPDPubId", input.Publisher.ID, "site-pub-id-copied")
	assert.JSONEq(t, `{"a": "inputPubExt", "b": 1}`, string(inputPublisher.Ext), "site-pub-ext-input")
	assert.JSONEq(t, `{"a": "FPDPubExt", "b": 2}`, string(input.Publisher.Ext), "site-pub-ext-copied")

	assert.Equal(t, "InContentId", inputContent.ID, "site-content-id-input")
	assert.Equal(t, "FPDContentId", input.Content.ID, "site-content-id-copied")
	assert.JSONEq(t, `{"a": "inputContentExt", "b": 1}`, string(inputContent.Ext), "site-content-ext-input")
	assert.JSONEq(t, `{"a": "FPDContentExt", "b": 2}`, string(input.Content.Ext), "site-content-ext-copied")

	assert.Equal(t, "InProducerId", inputContent.Producer.ID, "site-content-producer-id-input")
	assert.Equal(t, "FPDProducerId", input.Content.Producer.ID, "site-content-producer-id-copied")
	assert.JSONEq(t, `{"a": "inputProducerExt", "b": 1}`, string(inputContent.Producer.Ext), "site-content-producer-ext-input")
	assert.JSONEq(t, `{"a": "FPDProducerExt", "b": 2}`, string(input.Content.Producer.Ext), "site-content-producer-ext-copied")

	assert.Equal(t, "InNetworkId", inputContent.Network.ID, "site-content-network-id-input")
	assert.Equal(t, "FPDNetworkId", input.Content.Network.ID, "site-content-network-id-copied")
	assert.JSONEq(t, `{"a": "inputNetworkExt", "b": 1}`, string(inputContent.Network.Ext), "site-content-network-ext-input")
	assert.JSONEq(t, `{"a": "FPDNetworkExt", "b": 2}`, string(input.Content.Network.Ext), "site-content-network-ext-copied")

	assert.Equal(t, "InChannelId", inputContent.Channel.ID, "site-content-channel-id-input")
	assert.Equal(t, "FPDChannelId", input.Content.Channel.ID, "site-content-channel-id-copied")
	assert.JSONEq(t, `{"a": "inputChannelExt", "b": 1}`, string(inputContent.Channel.Ext), "site-content-channel-ext-input")
	assert.JSONEq(t, `{"a": "FPDChannelExt", "b": 2}`, string(input.Content.Channel.Ext), "site-content-channel-ext-copied")
}

var (
	userFPD = []byte(`
{
  "id": "2",
  "geo": {
    "ext": {
      "b": 100,
      "c": 3
    }
  }
}
`)

	fpdWithPublisherAndContent = []byte(`
{
  "id": "FPDID",
  "ext": {"a": "FPDExt", "b": 2},
  "publisher": {
    "id": "FPDPubId",
    "ext": {"a": "FPDPubExt", "b": 2}
  },
  "content": {
    "id": "FPDContentId",
    "ext": {"a": "FPDContentExt", "b": 2},
    "producer": {
      "id": "FPDProducerId",
      "ext": {"a": "FPDProducerExt", "b": 2}
    },
    "network": {
      "id": "FPDNetworkId",
      "ext": {"a": "FPDNetworkExt", "b": 2}
    },
    "channel": {
      "id": "FPDChannelId",
      "ext": {"a": "FPDChannelExt", "b": 2}
    }
  }
}
`)

	user = []byte(`
{
  "id": "2",
  "yob": 2000,
  "geo": {
    "city": "LA",
    "ext": {
      "b": 100,
      "c": 3
    }
  }
}
`)
)
