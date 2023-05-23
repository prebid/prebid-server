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

// fails
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

					// fmt.Println("expected:")
					// fmt.Println(string(expectedAppExt))

					// fmt.Println("\nhave:")
					// fmt.Println(string(resAppExt))

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

// fails
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

// func TestResolveUser(t *testing.T) {

// 	fpdConfigUser := make(map[string]json.RawMessage, 0)
// 	fpdConfigUser["id"] = []byte(`"fpdConfigUserId"`)
// 	fpdConfigUser[yobKey] = []byte(`1980`)
// 	fpdConfigUser[genderKey] = []byte(`"M"`)
// 	fpdConfigUser[keywordsKey] = []byte(`"fpdConfigUserKeywords"`)
// 	fpdConfigUser[dataKey] = []byte(`[{"id":"UserDataId1", "name":"UserDataName1"}, {"id":"UserDataId2", "name":"UserDataName2"}]`)
// 	fpdConfigUser["ext"] = []byte(`{"data":{"fpdConfigUserExt": 123}}`)

// 	bidRequestUser := &openrtb2.User{
// 		ID:       "bidRequestUserId",
// 		Yob:      1990,
// 		Gender:   "F",
// 		Keywords: "bidRequestUserKeywords",
// 	}

// 	globalFPD := make(map[string][]byte, 0)
// 	globalFPD[userKey] = []byte(`{"globalFPDUserData": "globalFPDUserDataValue"}`)

// 	openRtbGlobalFPD := make(map[string][]openrtb2.Data, 0)
// 	openRtbGlobalFPD[userDataKey] = []openrtb2.Data{
// 		{ID: "openRtbGlobalFPDUserDataId1", Name: "openRtbGlobalFPDUserDataName1"},
// 		{ID: "openRtbGlobalFPDUserDataId2", Name: "openRtbGlobalFPDUserDataName2"},
// 	}

// 	expectedUser := &openrtb2.User{
// 		ID:       "bidRequestUserId",
// 		Yob:      1980,
// 		Gender:   "M",
// 		Keywords: "fpdConfigUserKeywords",
// 		Data: []openrtb2.Data{
// 			{ID: "openRtbGlobalFPDUserDataId1", Name: "openRtbGlobalFPDUserDataName1"},
// 			{ID: "openRtbGlobalFPDUserDataId2", Name: "openRtbGlobalFPDUserDataName2"},
// 			{ID: "UserDataId1", Name: "UserDataName1"},
// 			{ID: "UserDataId2", Name: "UserDataName2"},
// 		},
// 	}

// 	testCases := []struct {
// 		description       string
// 		bidRequestUserExt []byte
// 		expectedUserExt   string
// 	}{
// 		{
// 			description:       "bid request user.ext is nil",
// 			bidRequestUserExt: nil,
// 			expectedUserExt: `{"data":{
// 									"fpdConfigUserExt":123,
// 									"globalFPDUserData":"globalFPDUserDataValue",
// 									"id":"fpdConfigUserId"
// 									}
// 							}`,
// 		},
// 		{
// 			description:       "bid request user.ext is not nil",
// 			bidRequestUserExt: []byte(`{"bidRequestUserExt": 1234}`),
// 			expectedUserExt: `{"data":{
// 									"fpdConfigUserExt":123,
// 									"globalFPDUserData":"globalFPDUserDataValue",
// 									"id":"fpdConfigUserId"
// 									},
// 								"bidRequestUserExt":1234
// 							}`,
// 		},
// 	}

// 	for _, test := range testCases {
// 		bidRequestUser.Ext = test.bidRequestUserExt

// 		fpdConfigUser := make(map[string]json.RawMessage, 0)
// 		fpdConfigUser["id"] = []byte(`"fpdConfigUserId"`)
// 		fpdConfigUser[yobKey] = []byte(`1980`)
// 		fpdConfigUser[genderKey] = []byte(`"M"`)
// 		fpdConfigUser[keywordsKey] = []byte(`"fpdConfigUserKeywords"`)
// 		fpdConfigUser[dataKey] = []byte(`[{"id":"UserDataId1", "name":"UserDataName1"}, {"id":"UserDataId2", "name":"UserDataName2"}]`)
// 		fpdConfigUser["ext"] = []byte(`{"data":{"fpdConfigUserExt": 123}}`)
// 		fpdConfig := &openrtb_ext.ORTB2{User: fpdConfigUser}

// 		resultUser, err := resolveUser(fpdConfig, bidRequestUser, globalFPD, openRtbGlobalFPD, "appnexus")
// 		assert.NoError(t, err, "No error should be returned")

// 		assert.JSONEq(t, test.expectedUserExt, string(resultUser.Ext), "Result user.Ext is incorrect")
// 		resultUser.Ext = nil
// 		assert.Equal(t, expectedUser, resultUser, "Result user is incorrect")
// 	}

// }

func TestResolveUserNilValues(t *testing.T) {
	resultUser, err := resolveUser(nil, nil, nil, nil, "appnexus")
	assert.NoError(t, err, "No error should be returned")
	assert.Nil(t, resultUser, "Result user should be nil")
}

// func TestMergeUsers(t *testing.T) {

// 	originalUser := &openrtb2.User{
// 		ID:       "bidRequestUserId",
// 		Yob:      1980,
// 		Gender:   "M",
// 		Keywords: "fpdConfigUserKeywords",
// 		Data: []openrtb2.Data{
// 			{ID: "openRtbGlobalFPDUserDataId1", Name: "openRtbGlobalFPDUserDataName1"},
// 			{ID: "openRtbGlobalFPDUserDataId2", Name: "openRtbGlobalFPDUserDataName2"},
// 		},
// 		Ext: []byte(`{"bidRequestUserExt": 1234}`),
// 	}
// 	fpdConfigUser := make(map[string]json.RawMessage, 0)
// 	fpdConfigUser["id"] = []byte(`"fpdConfigUserId"`)
// 	fpdConfigUser[yobKey] = []byte(`1980`)
// 	fpdConfigUser[genderKey] = []byte(`"M"`)
// 	fpdConfigUser[keywordsKey] = []byte(`"fpdConfigUserKeywords"`)
// 	fpdConfigUser[dataKey] = []byte(`[{"id":"UserDataId1", "name":"UserDataName1"}, {"id":"UserDataId2", "name":"UserDataName2"}]`)
// 	fpdConfigUser["ext"] = []byte(`{"data":{"fpdConfigUserExt": 123}}`)

// 	resultUser, err := mergeUsers(originalUser, fpdConfigUser)
// 	assert.NoError(t, err, "No error should be returned")

// 	expectedUserExt := `{"bidRequestUserExt":1234,
// 						 "data":{
// 							"fpdConfigUserExt":123,
// 							"id":"fpdConfigUserId"}
// 						 }`
// 	assert.JSONEq(t, expectedUserExt, string(resultUser.Ext), "Result user.Ext is incorrect")
// 	resultUser.Ext = nil

// 	expectedUser := openrtb2.User{
// 		ID:       "bidRequestUserId",
// 		Yob:      1980,
// 		Gender:   "M",
// 		Keywords: "fpdConfigUserKeywords",
// 		Data: []openrtb2.Data{
// 			{ID: "openRtbGlobalFPDUserDataId1", Name: "openRtbGlobalFPDUserDataName1"},
// 			{ID: "openRtbGlobalFPDUserDataId2", Name: "openRtbGlobalFPDUserDataName2"},
// 			{ID: "UserDataId1", Name: "UserDataName1"},
// 			{ID: "UserDataId2", Name: "UserDataName2"},
// 		},
// 	}
// 	assert.Equal(t, expectedUser, resultUser, "Result user is incorrect")
// }

// func TestResolveSite(t *testing.T) {

// 	fpdConfigSite := make(map[string]json.RawMessage, 0)
// 	fpdConfigSite["id"] = []byte(`"fpdConfigSiteId"`)
// 	fpdConfigSite[keywordsKey] = []byte(`"fpdConfigSiteKeywords"`)
// 	fpdConfigSite[nameKey] = []byte(`"fpdConfigSiteName"`)
// 	fpdConfigSite[pageKey] = []byte(`"fpdConfigSitePage"`)
// 	fpdConfigSite[dataKey] = []byte(`[{"id":"SiteDataId1", "name":"SiteDataName1"}, {"id":"SiteDataId2", "name":"SiteDataName2"}]`)
// 	fpdConfigSite["ext"] = []byte(`{"data":{"fpdConfigSiteExt": 123}}`)

// 	bidRequestSite := &openrtb2.Site{
// 		ID:       "bidRequestSiteId",
// 		Keywords: "bidRequestSiteKeywords",
// 		Name:     "bidRequestSiteName",
// 		Page:     "bidRequestSitePage",
// 		Content: &openrtb2.Content{
// 			ID:      "bidRequestSiteContentId",
// 			Episode: 4,
// 			Data: []openrtb2.Data{
// 				{ID: "bidRequestSiteContentDataId1", Name: "bidRequestSiteContentDataName1"},
// 				{ID: "bidRequestSiteContentDataId2", Name: "bidRequestSiteContentDataName2"},
// 			},
// 		},
// 	}

// 	globalFPD := make(map[string][]byte, 0)
// 	globalFPD[siteKey] = []byte(`{"globalFPDSiteData": "globalFPDSiteDataValue"}`)

// 	openRtbGlobalFPD := make(map[string][]openrtb2.Data, 0)
// 	openRtbGlobalFPD[siteContentDataKey] = []openrtb2.Data{
// 		{ID: "openRtbGlobalFPDSiteContentDataId1", Name: "openRtbGlobalFPDSiteContentDataName1"},
// 		{ID: "openRtbGlobalFPDSiteContentDataId2", Name: "openRtbGlobalFPDSiteContentDataName2"},
// 	}

// 	expectedSite := &openrtb2.Site{
// 		ID:       "bidRequestSiteId",
// 		Keywords: "fpdConfigSiteKeywords",
// 		Name:     "bidRequestSiteName",
// 		Page:     "bidRequestSitePage",
// 		Content: &openrtb2.Content{
// 			ID:      "bidRequestSiteContentId",
// 			Episode: 4,
// 			Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDSiteContentDataId1", Name: "openRtbGlobalFPDSiteContentDataName1"},
// 				{ID: "openRtbGlobalFPDSiteContentDataId2", Name: "openRtbGlobalFPDSiteContentDataName2"},
// 			},
// 		},
// 	}

// 	testCases := []struct {
// 		description       string
// 		bidRequestSiteExt []byte
// 		expectedSiteExt   string
// 		siteContentNil    bool
// 	}{
// 		{
// 			description:       "bid request site.ext is nil",
// 			bidRequestSiteExt: nil,
// 			expectedSiteExt: `{"data":{
// 									"data":[
// 										{"id":"SiteDataId1","name":"SiteDataName1"},
// 										{"id":"SiteDataId2","name":"SiteDataName2"}
// 									],
// 									"fpdConfigSiteExt":123,
// 									"globalFPDSiteData":"globalFPDSiteDataValue",
// 									"id":"fpdConfigSiteId"
// 									}
// 							}`,
// 			siteContentNil: false,
// 		},
// 		{
// 			description:       "bid request site.ext is not nil",
// 			bidRequestSiteExt: []byte(`{"bidRequestSiteExt": 1234}`),
// 			expectedSiteExt: `{"data":{
// 									"data":[
// 										{"id":"SiteDataId1","name":"SiteDataName1"},
// 										{"id":"SiteDataId2","name":"SiteDataName2"}
// 									],
// 									"fpdConfigSiteExt":123,
// 									"globalFPDSiteData":"globalFPDSiteDataValue",
// 									"id":"fpdConfigSiteId"
// 									},
// 								"bidRequestSiteExt":1234
// 							}`,
// 			siteContentNil: false,
// 		},
// 		{
// 			description:       "bid request site.content.data is nil ",
// 			bidRequestSiteExt: []byte(`{"bidRequestSiteExt": 1234}`),
// 			expectedSiteExt: `{"data":{
// 									"data":[
// 										{"id":"SiteDataId1","name":"SiteDataName1"},
// 										{"id":"SiteDataId2","name":"SiteDataName2"}
// 									],
// 									"fpdConfigSiteExt":123,
// 									"globalFPDSiteData":"globalFPDSiteDataValue",
// 									"id":"fpdConfigSiteId"
// 									},
// 								"bidRequestSiteExt":1234
// 							}`,
// 			siteContentNil: true,
// 		},
// 	}

// 	for _, test := range testCases {
// 		if test.siteContentNil {
// 			bidRequestSite.Content = nil
// 			expectedSite.Content = &openrtb2.Content{Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDSiteContentDataId1", Name: "openRtbGlobalFPDSiteContentDataName1"},
// 				{ID: "openRtbGlobalFPDSiteContentDataId2", Name: "openRtbGlobalFPDSiteContentDataName2"},
// 			}}
// 		}

// 		bidRequestSite.Ext = test.bidRequestSiteExt

// 		fpdConfigSite := make(map[string]json.RawMessage, 0)
// 		fpdConfigSite["id"] = []byte(`"fpdConfigSiteId"`)
// 		fpdConfigSite[keywordsKey] = []byte(`"fpdConfigSiteKeywords"`)
// 		fpdConfigSite[dataKey] = []byte(`[{"id":"SiteDataId1", "name":"SiteDataName1"}, {"id":"SiteDataId2", "name":"SiteDataName2"}]`)
// 		fpdConfigSite["ext"] = []byte(`{"data":{"fpdConfigSiteExt": 123}}`)
// 		fpdConfig := &openrtb_ext.ORTB2{Site: fpdConfigSite}

// 		resultSite, err := resolveSite(fpdConfig, bidRequestSite, globalFPD, openRtbGlobalFPD, "appnexus")
// 		assert.NoError(t, err, "No error should be returned")

// 		assert.JSONEq(t, test.expectedSiteExt, string(resultSite.Ext), "Result site.Ext is incorrect")
// 		resultSite.Ext = nil
// 		assert.Equal(t, expectedSite, resultSite, "Result site is incorrect")
// 	}

// }

func TestResolveSiteNilValues(t *testing.T) {
	resultSite, err := resolveSite(nil, nil, nil, nil, "appnexus")
	assert.NoError(t, err, "No error should be returned")
	assert.Nil(t, resultSite, "Result site should be nil")
}

// func TestResolveSiteBadInput(t *testing.T) {
// 	fpdConfigSite := make(map[string]json.RawMessage, 0)
// 	fpdConfigSite["id"] = []byte(`"fpdConfigSiteId"`)
// 	fpdConfig := &openrtb_ext.ORTB2{Site: fpdConfigSite}

// 	resultSite, err := resolveSite(fpdConfig, nil, nil, nil, "appnexus")
// 	assert.Error(t, err, "Error should be returned")
// 	assert.Equal(t, "incorrect First Party Data for bidder appnexus: Site object is not defined in request, but defined in FPD config", err.Error(), "Incorrect error message")
// 	assert.Nil(t, resultSite, "Result site should be nil")
// }

// func TestMergeSites(t *testing.T) {

// 	originalSite := &openrtb2.Site{
// 		ID:         "bidRequestSiteId",
// 		Keywords:   "bidRequestSiteKeywords",
// 		Page:       "bidRequestSitePage",
// 		Name:       "bidRequestSiteName",
// 		Domain:     "bidRequestSiteDomain",
// 		Cat:        []string{"books1", "magazines1"},
// 		SectionCat: []string{"books2", "magazines2"},
// 		PageCat:    []string{"books3", "magazines3"},
// 		Search:     "bidRequestSiteSearch",
// 		Ref:        "bidRequestSiteRef",
// 		Content: &openrtb2.Content{
// 			Title: "bidRequestSiteContentTitle",
// 			Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDSiteDataId1", Name: "openRtbGlobalFPDSiteDataName1"},
// 				{ID: "openRtbGlobalFPDSiteDataId2", Name: "openRtbGlobalFPDSiteDataName2"},
// 			},
// 		},
// 		Ext: []byte(`{"bidRequestSiteExt": 1234}`),
// 	}
// 	fpdConfigSite := make(map[string]json.RawMessage, 0)
// 	fpdConfigSite["id"] = []byte(`"fpdConfigSiteId"`)
// 	fpdConfigSite[keywordsKey] = []byte(`"fpdConfigSiteKeywords"`)
// 	fpdConfigSite[pageKey] = []byte(`"fpdConfigSitePage"`)
// 	fpdConfigSite[nameKey] = []byte(`"fpdConfigSiteName"`)
// 	fpdConfigSite[domainKey] = []byte(`"fpdConfigSiteDomain"`)
// 	fpdConfigSite[catKey] = []byte(`["cars1", "auto1"]`)
// 	fpdConfigSite[sectionCatKey] = []byte(`["cars2", "auto2"]`)
// 	fpdConfigSite[pageCatKey] = []byte(`["cars3", "auto3"]`)
// 	fpdConfigSite[searchKey] = []byte(`"fpdConfigSiteSearch"`)
// 	fpdConfigSite[refKey] = []byte(`"fpdConfigSiteRef"`)
// 	fpdConfigSite[dataKey] = []byte(`[{"id":"SiteDataId1", "name":"SiteDataName1"}, {"id":"SiteDataId2", "name":"SiteDataName2"}]`)
// 	fpdConfigSite["ext"] = []byte(`{"data":{"fpdConfigSiteExt": 123}}`)

// 	resultSite, err := mergeSites(originalSite, fpdConfigSite, "appnexus")
// 	assert.NoError(t, err, "No error should be returned")

// 	expectedSiteExt := `{"bidRequestSiteExt":1234,
// 						 "data":{
// 							"data":[
// 								{"id":"SiteDataId1","name":"SiteDataName1"},
// 								{"id":"SiteDataId2","name":"SiteDataName2"}],
// 							"fpdConfigSiteExt":123,
// 							"id":"fpdConfigSiteId"}
// 						 }`
// 	assert.JSONEq(t, expectedSiteExt, string(resultSite.Ext), "Result user.Ext is incorrect")
// 	resultSite.Ext = nil

// 	expectedSite := openrtb2.Site{
// 		ID:         "bidRequestSiteId",
// 		Keywords:   "fpdConfigSiteKeywords",
// 		Page:       "fpdConfigSitePage",
// 		Name:       "fpdConfigSiteName",
// 		Domain:     "fpdConfigSiteDomain",
// 		Cat:        []string{"cars1", "auto1"},
// 		SectionCat: []string{"cars2", "auto2"},
// 		PageCat:    []string{"cars3", "auto3"},
// 		Search:     "fpdConfigSiteSearch",
// 		Ref:        "fpdConfigSiteRef",
// 		Content: &openrtb2.Content{
// 			Title: "bidRequestSiteContentTitle",
// 			Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDSiteDataId1", Name: "openRtbGlobalFPDSiteDataName1"},
// 				{ID: "openRtbGlobalFPDSiteDataId2", Name: "openRtbGlobalFPDSiteDataName2"},
// 			},
// 		},
// 		Ext: nil,
// 	}
// 	assert.Equal(t, expectedSite, resultSite, "Result user is incorrect")
// }

// func TestResolveApp(t *testing.T) {

// 	fpdConfigApp := make(map[string]json.RawMessage, 0)
// 	fpdConfigApp["id"] = []byte(`"fpdConfigAppId"`)
// 	fpdConfigApp[keywordsKey] = []byte(`"fpdConfigAppKeywords"`)
// 	fpdConfigApp[nameKey] = []byte(`"fpdConfigAppName"`)
// 	fpdConfigApp[bundleKey] = []byte(`"fpdConfigAppBundle"`)
// 	fpdConfigApp[dataKey] = []byte(`[{"id":"AppDataId1", "name":"AppDataName1"}, {"id":"AppDataId2", "name":"AppDataName2"}]`)
// 	fpdConfigApp["ext"] = []byte(`{"data":{"fpdConfigAppExt": 123}}`)

// 	bidRequestApp := &openrtb2.App{
// 		ID:       "bidRequestAppId",
// 		Keywords: "bidRequestAppKeywords",
// 		Name:     "bidRequestAppName",
// 		Bundle:   "bidRequestAppBundle",
// 		Content: &openrtb2.Content{
// 			ID:      "bidRequestAppContentId",
// 			Episode: 4,
// 			Data: []openrtb2.Data{
// 				{ID: "bidRequestAppContentDataId1", Name: "bidRequestAppContentDataName1"},
// 				{ID: "bidRequestAppContentDataId2", Name: "bidRequestAppContentDataName2"},
// 			},
// 		},
// 	}

// 	globalFPD := make(map[string][]byte, 0)
// 	globalFPD[appKey] = []byte(`{"globalFPDAppData": "globalFPDAppDataValue"}`)

// 	openRtbGlobalFPD := make(map[string][]openrtb2.Data, 0)
// 	openRtbGlobalFPD[appContentDataKey] = []openrtb2.Data{
// 		{ID: "openRtbGlobalFPDAppContentDataId1", Name: "openRtbGlobalFPDAppContentDataName1"},
// 		{ID: "openRtbGlobalFPDAppContentDataId2", Name: "openRtbGlobalFPDAppContentDataName2"},
// 	}

// 	expectedApp := &openrtb2.App{
// 		ID:       "bidRequestAppId",
// 		Keywords: "fpdConfigAppKeywords",
// 		Name:     "bidRequestAppName",
// 		Bundle:   "bidRequestAppBundle",
// 		Content: &openrtb2.Content{
// 			ID:      "bidRequestAppContentId",
// 			Episode: 4,
// 			Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDAppContentDataId1", Name: "openRtbGlobalFPDAppContentDataName1"},
// 				{ID: "openRtbGlobalFPDAppContentDataId2", Name: "openRtbGlobalFPDAppContentDataName2"},
// 			},
// 		},
// 	}

// 	testCases := []struct {
// 		description      string
// 		bidRequestAppExt []byte
// 		expectedAppExt   string
// 		appContentNil    bool
// 	}{
// 		{
// 			description:      "bid request app.ext is nil",
// 			bidRequestAppExt: nil,
// 			expectedAppExt: `{"data":{
// 									"data":[
// 										{"id":"AppDataId1","name":"AppDataName1"},
// 										{"id":"AppDataId2","name":"AppDataName2"}
// 									],
// 									"fpdConfigAppExt":123,
// 									"globalFPDAppData":"globalFPDAppDataValue",
// 									"id":"fpdConfigAppId"
// 									}
// 							}`,
// 			appContentNil: false,
// 		},
// 		{
// 			description:      "bid request app.ext is not nil",
// 			bidRequestAppExt: []byte(`{"bidRequestAppExt": 1234}`),
// 			expectedAppExt: `{"data":{
// 									"data":[
// 										{"id":"AppDataId1","name":"AppDataName1"},
// 										{"id":"AppDataId2","name":"AppDataName2"}
// 									],
// 									"fpdConfigAppExt":123,
// 									"globalFPDAppData":"globalFPDAppDataValue",
// 									"id":"fpdConfigAppId"
// 									},
// 								"bidRequestAppExt":1234
// 							}`,
// 			appContentNil: false,
// 		},
// 		{
// 			description:      "bid request app.content.data is nil ",
// 			bidRequestAppExt: []byte(`{"bidRequestAppExt": 1234}`),
// 			expectedAppExt: `{"data":{
// 									"data":[
// 										{"id":"AppDataId1","name":"AppDataName1"},
// 										{"id":"AppDataId2","name":"AppDataName2"}
// 									],
// 									"fpdConfigAppExt":123,
// 									"globalFPDAppData":"globalFPDAppDataValue",
// 									"id":"fpdConfigAppId"
// 									},
// 								"bidRequestAppExt":1234
// 							}`,
// 			appContentNil: true,
// 		},
// 	}

// 	for _, test := range testCases {
// 		if test.appContentNil {
// 			bidRequestApp.Content = nil
// 			expectedApp.Content = &openrtb2.Content{Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDAppContentDataId1", Name: "openRtbGlobalFPDAppContentDataName1"},
// 				{ID: "openRtbGlobalFPDAppContentDataId2", Name: "openRtbGlobalFPDAppContentDataName2"},
// 			}}
// 		}

// 		bidRequestApp.Ext = test.bidRequestAppExt

// 		fpdConfigApp := make(map[string]json.RawMessage, 0)
// 		fpdConfigApp["id"] = []byte(`"fpdConfigAppId"`)
// 		fpdConfigApp[keywordsKey] = []byte(`"fpdConfigAppKeywords"`)
// 		fpdConfigApp[dataKey] = []byte(`[{"id":"AppDataId1", "name":"AppDataName1"}, {"id":"AppDataId2", "name":"AppDataName2"}]`)
// 		fpdConfigApp["ext"] = []byte(`{"data":{"fpdConfigAppExt": 123}}`)
// 		fpdConfig := &openrtb_ext.ORTB2{App: fpdConfigApp}

// 		resultApp, err := resolveApp(fpdConfig, bidRequestApp, globalFPD, openRtbGlobalFPD, "appnexus")
// 		assert.NoError(t, err, "No error should be returned")

// 		assert.JSONEq(t, test.expectedAppExt, string(resultApp.Ext), "Result app.Ext is incorrect")
// 		resultApp.Ext = nil
// 		assert.Equal(t, expectedApp, resultApp, "Result app is incorrect")
// 	}

// }

func TestResolveAppNilValues(t *testing.T) {
	resultApp, err := resolveApp(nil, nil, nil, nil, "appnexus")
	assert.NoError(t, err, "No error should be returned")
	assert.Nil(t, resultApp, "Result app should be nil")
}

// func TestResolveAppBadInput(t *testing.T) {
// 	fpdConfigApp := make(map[string]json.RawMessage, 0)
// 	fpdConfigApp["id"] = []byte(`"fpdConfigAppId"`)
// 	fpdConfig := &openrtb_ext.ORTB2{App: fpdConfigApp}

// 	resultApp, err := resolveApp(fpdConfig, nil, nil, nil, "appnexus")
// 	assert.Error(t, err, "Error should be returned")
// 	assert.Equal(t, "incorrect First Party Data for bidder appnexus: App object is not defined in request, but defined in FPD config", err.Error(), "Incorrect error message")
// 	assert.Nil(t, resultApp, "Result app should be nil")
// }

// func TestMergeApps(t *testing.T) {

// 	originalApp := &openrtb2.App{
// 		ID:         "bidRequestAppId",
// 		Keywords:   "bidRequestAppKeywords",
// 		Name:       "bidRequestAppName",
// 		Domain:     "bidRequestAppDomain",
// 		Bundle:     "bidRequestAppBundle",
// 		StoreURL:   "bidRequestAppStoreUrl",
// 		Ver:        "bidRequestAppVer",
// 		Cat:        []string{"books1", "magazines1"},
// 		SectionCat: []string{"books2", "magazines2"},
// 		PageCat:    []string{"books3", "magazines3"},
// 		Content: &openrtb2.Content{
// 			Title: "bidRequestAppContentTitle",
// 			Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDAppDataId1", Name: "openRtbGlobalFPDAppDataName1"},
// 				{ID: "openRtbGlobalFPDAppDataId2", Name: "openRtbGlobalFPDAppDataName2"},
// 			},
// 		},
// 		Ext: []byte(`{"bidRequestAppExt": 1234}`),
// 	}
// 	fpdConfigApp := make(map[string]json.RawMessage, 0)
// 	fpdConfigApp["id"] = []byte(`"fpdConfigAppId"`)
// 	fpdConfigApp[keywordsKey] = []byte(`"fpdConfigAppKeywords"`)
// 	fpdConfigApp[nameKey] = []byte(`"fpdConfigAppName"`)
// 	fpdConfigApp[domainKey] = []byte(`"fpdConfigAppDomain"`)
// 	fpdConfigApp[bundleKey] = []byte(`"fpdConfigAppBundle"`)
// 	fpdConfigApp[storeUrlKey] = []byte(`"fpdConfigAppStoreUrl"`)
// 	fpdConfigApp[verKey] = []byte(`"fpdConfigAppVer"`)
// 	fpdConfigApp[catKey] = []byte(`["cars1", "auto1"]`)
// 	fpdConfigApp[sectionCatKey] = []byte(`["cars2", "auto2"]`)
// 	fpdConfigApp[pageCatKey] = []byte(`["cars3", "auto3"]`)
// 	fpdConfigApp[dataKey] = []byte(`[{"id":"AppDataId1", "name":"AppDataName1"}, {"id":"AppDataId2", "name":"AppDataName2"}]`)
// 	fpdConfigApp["ext"] = []byte(`{"data":{"fpdConfigAppExt": 123}}`)

// 	resultApp, err := mergeApps(originalApp, fpdConfigApp)
// 	assert.NoError(t, err, "No error should be returned")

// 	expectedAppExt := `{"bidRequestAppExt":1234,
// 						 "data":{
// 							"data":[
// 								{"id":"AppDataId1","name":"AppDataName1"},
// 								{"id":"AppDataId2","name":"AppDataName2"}],
// 							"fpdConfigAppExt":123,
// 							"id":"fpdConfigAppId"}
// 						 }`
// 	assert.JSONEq(t, expectedAppExt, string(resultApp.Ext), "Result user.Ext is incorrect")
// 	resultApp.Ext = nil

// 	expectedApp := openrtb2.App{
// 		ID:         "bidRequestAppId",
// 		Keywords:   "fpdConfigAppKeywords",
// 		Name:       "fpdConfigAppName",
// 		Domain:     "fpdConfigAppDomain",
// 		Bundle:     "fpdConfigAppBundle",
// 		Ver:        "fpdConfigAppVer",
// 		StoreURL:   "fpdConfigAppStoreUrl",
// 		Cat:        []string{"cars1", "auto1"},
// 		SectionCat: []string{"cars2", "auto2"},
// 		PageCat:    []string{"cars3", "auto3"},
// 		Content: &openrtb2.Content{
// 			Title: "bidRequestAppContentTitle",
// 			Data: []openrtb2.Data{
// 				{ID: "openRtbGlobalFPDAppDataId1", Name: "openRtbGlobalFPDAppDataName1"},
// 				{ID: "openRtbGlobalFPDAppDataId2", Name: "openRtbGlobalFPDAppDataName2"},
// 			},
// 		},
// 		Ext: nil,
// 	}
// 	assert.Equal(t, expectedApp, resultApp, "Result user is incorrect")
// }

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
				assert.NoError(t, err, "error")
				assert.Equal(t, test.expectedUser, test.givenUser, "user")
			} else {
				assert.EqualError(t, err, test.expectedErr, "error")
			}
		})
	}
}

func TestMergeUserMemoryProtection(t *testing.T) {
	inputGeo := &openrtb2.Geo{
		Ext: json.RawMessage(`{"a":1,"b":2}`),
	}
	input := openrtb2.User{
		ID:  "1",
		Geo: inputGeo,
	}

	firstPartyData := json.RawMessage(`{"id":"2","geo":{"ext":{"b":100,"c":3}}}`)

	err := mergeUser(&input, firstPartyData)
	assert.NoError(t, err)

	// Input user object is expected to be a copy. Changes are ok.
	assert.Equal(t, "2", input.ID)

	// Nested objects must be copied before changes.
	assert.Equal(t, json.RawMessage(`{"a":1,"b":2}`), inputGeo.Ext, "geo-input")
	assert.Equal(t, json.RawMessage(`{"a":1,"b":100,"c":3}`), input.Geo.Ext, "geo-copied")
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

// todo:
// add: app memory protect test
// add: site memory protect test
