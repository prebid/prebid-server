package firstpartydata

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"testing"
)

func TestExtractGlobalFPD(t *testing.T) {

	testCases := []struct {
		description string
		input       []byte
		output      []byte
		expectedFpd map[string][]byte
	}{
		{
			description: "Site, app and user data present",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
				  "ext":{
				   "data": {"somesitefpd": "sitefpdDataTest"},
				   "amp": 1
				  }
  				},
  				"user": {
  				  "id": "reqUserID",
  				  "yob": 1982,
  				  "gender": "M",
				  "ext":{
				  	"data": {"someuserfpd": "userfpdDataTest"}
				  }
  				},
  				"app": {
  				  "id": "appId",
  				  "data": 123,
  				  "ext": {
				     "data": {"someappfpd": "appfpdDataTest"}
				  }
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
				  "ext": {
					"amp": 1
				  }
  				},
  				"user": {
  				  "id": "reqUserID",
  				  "yob": 1982,
  				  "gender": "M"
  				},
  				"app": {
  				  "id": "appId",
  				  "data": 123
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			expectedFpd: map[string][]byte{
				"site": []byte(`{"somesitefpd": "sitefpdDataTest"}`),
				"user": []byte(`{"someuserfpd": "userfpdDataTest"}`),
				"app":  []byte(`{"someappfpd": "appfpdDataTest"}`),
			},
		},
		{
			description: "App FPD only present",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  }
  				},
  				"app": {
  				  "id": "appId",
  				  "ext": {
					"data": {"someappfpd": "appfpdDataTest"}
                  }
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  }
  				},
  				"app": {
  				  "id": "appId",
				  "ext": {}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			expectedFpd: map[string][]byte{
				"app":  []byte(`{"someappfpd": "appfpdDataTest"}`),
				"user": nil,
				"site": nil,
			},
		},
		{
			description: "User FPD only present",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  }
  				},
  				"user": {
  				  "id": "userId",
  				  "ext": {
					"data": {"someuserfpd": "userfpdDataTest"}
                  }
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  }
  				},
  				"user": {
  				  "id": "appId",
				  "ext": {}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			expectedFpd: map[string][]byte{
				"app":  nil,
				"user": []byte(`{"someuserfpd": "userfpdDataTest"}`),
				"site": nil,
			},
		},
		{
			description: "No FPD present in req",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  }
  				},
  				"app": {
  				  "id": "appId",
  				  "ext": {
                  }
  				},
			  	"user": {
  				  "id": "userId",
				  "ext": {}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  }
  				},
  				"app": {
  				  "id": "appId",
				  "ext": {}
  				},
				"user": {
  				  "id": "userId",
				  "ext": {}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			expectedFpd: map[string][]byte{
				"app":  nil,
				"user": nil,
				"site": nil,
			},
		},
		{
			description: "Site FPD only present",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
				  "ext": {
					"data": {"someappfpd": true},
					"amp": 1
                  }
  				},
  				"app": {
  				  "id": "appId"
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
                 "ext": {
					"amp": 1
                  }
  				},
  				"app": {
  				  "id": "appId"
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			expectedFpd: map[string][]byte{
				"app":  nil,
				"user": nil,
				"site": []byte(`{"someappfpd": true}`),
			},
		},
	}
	for _, test := range testCases {
		var inputTestReq openrtb_ext.RequestWrapper
		err := json.Unmarshal(test.input, &inputTestReq)
		assert.NoError(t, err, "Error should be nil")

		fpd, err := ExtractGlobalFPD(&inputTestReq)
		inputTestReq.RebuildRequest()
		assert.NoError(t, err, "Error should be nil")

		var outputTestReq openrtb_ext.RequestWrapper
		err = json.Unmarshal(test.output, &outputTestReq)
		assert.NoError(t, err, "Error should be nil")

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

	if specFiles, err := ioutil.ReadDir("./tests/extractbidderconfigfpd"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./tests/extractbidderconfigfpd/" + specFile.Name()

			fpdFile, err := loadFpdFile(fileName)
			if err != nil {
				t.Errorf("Unable to load file: %s", fileName)
			}
			var extReq openrtb_ext.ExtRequestPrebid
			err = json.Unmarshal(fpdFile.InputRequestData, &extReq)
			if err != nil {
				t.Errorf("Unable to unmarshal input request: %s", fileName)
			}
			reqExt := openrtb_ext.RequestExt{}
			reqExt.SetPrebid(&extReq)
			fpdData, err := ExtractBidderConfigFPD(&reqExt)

			if len(fpdFile.ValidationErrors) > 0 {
				assert.EqualError(t, err, fpdFile.ValidationErrors[0], "Incorrect first party data error message")
				continue
			}

			assert.Nil(t, reqExt.GetPrebid().BidderConfigs, "Bidder specific FPD config should be removed from request")

			assert.Nil(t, err, "No error should be returned")
			assert.Equal(t, len(fpdFile.BiddersFPD), len(fpdData), "Incorrect fpd data")

			for bidderName, bidderFPD := range fpdFile.BiddersFPD {

				if bidderFPD.Site != nil {
					resSite := fpdData[bidderName].Site
					for k, v := range bidderFPD.Site {
						assert.NotNil(t, resSite[k], "Property is not found in result site")
						assert.JSONEq(t, string(v), string(resSite[k]), "site is incorrect")
					}
				} else {
					assert.Nil(t, fpdData[bidderName].Site, "Result site should be also nil")
				}

				if bidderFPD.App != nil {
					resApp := fpdData[bidderName].App
					for k, v := range bidderFPD.App {
						assert.NotNil(t, resApp[k], "Property is not found in result app")
						assert.JSONEq(t, string(v), string(resApp[k]), "app is incorrect")
					}
				} else {
					assert.Nil(t, fpdData[bidderName].App, "Result app should be also nil")
				}

				if bidderFPD.User != nil {
					resUser := fpdData[bidderName].User
					for k, v := range bidderFPD.User {
						assert.NotNil(t, resUser[k], "Property is not found in result user")
						assert.JSONEq(t, string(v), string(resUser[k]), "site is incorrect")
					}
				} else {
					assert.Nil(t, fpdData[bidderName].User, "Result user should be also nil")
				}
			}
		}
	}
}

func TestResolveFPD(t *testing.T) {

	if specFiles, err := ioutil.ReadDir("./tests/resolvefpd"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./tests/resolvefpd/" + specFile.Name()

			fpdFile, err := loadFpdFile(fileName)
			if err != nil {
				t.Errorf("Unable to load file: %s", fileName)
			}

			var inputReq openrtb2.BidRequest
			err = json.Unmarshal(fpdFile.InputRequestData, &inputReq)
			if err != nil {
				t.Errorf("Unable to unmarshal input request: %s", fileName)
			}

			var inputReqCopy openrtb2.BidRequest
			err = json.Unmarshal(fpdFile.InputRequestData, &inputReqCopy)
			if err != nil {
				t.Errorf("Unable to unmarshal input request: %s", fileName)
			}

			var outputReq openrtb2.BidRequest
			err = json.Unmarshal(fpdFile.OutputRequestData, &outputReq)
			if err != nil {
				t.Errorf("Unable to unmarshal output request: %s", fileName)
			}

			reqExtFPD := make(map[string][]byte, 3)
			reqExtFPD["site"] = fpdFile.FirstPartyData["site"]
			reqExtFPD["app"] = fpdFile.FirstPartyData["app"]
			reqExtFPD["user"] = fpdFile.FirstPartyData["user"]

			reqFPD := make(map[string][]openrtb2.Data, 3)

			reqFPDSiteContentData := fpdFile.FirstPartyData[siteContentDataKey]
			if len(reqFPDSiteContentData) > 0 {
				var siteConData []openrtb2.Data
				err = json.Unmarshal(reqFPDSiteContentData, &siteConData)
				if err != nil {
					t.Errorf("Unable to unmarshal site.content.data: %s", fileName)
				}
				reqFPD[siteContentDataKey] = siteConData
			}

			reqFPDAppContentData := fpdFile.FirstPartyData[appContentDataKey]
			if len(reqFPDAppContentData) > 0 {
				var appConData []openrtb2.Data
				err = json.Unmarshal(reqFPDAppContentData, &appConData)
				if err != nil {
					t.Errorf("Unable to unmarshal app.content.data: %s", fileName)
				}
				reqFPD[appContentDataKey] = appConData
			}

			reqFPDUserData := fpdFile.FirstPartyData[userDataKey]
			if len(reqFPDUserData) > 0 {
				var userData []openrtb2.Data
				err = json.Unmarshal(reqFPDUserData, &userData)
				if err != nil {
					t.Errorf("Unable to unmarshal app.content.data: %s", fileName)
				}
				reqFPD[userDataKey] = userData
			}
			if fpdFile.BiddersFPD == nil {
				fpdFile.BiddersFPD = make(map[openrtb_ext.BidderName]*openrtb_ext.ORTB2)
				fpdFile.BiddersFPD["appnexus"] = &openrtb_ext.ORTB2{}
			}

			resultFPD, errL := ResolveFPD(&inputReq, fpdFile.BiddersFPD, reqExtFPD, reqFPD, []string{"appnexus"})

			if len(errL) == 0 {
				assert.Equal(t, inputReq, inputReqCopy, "Original request should not be modified")

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
				for i := range fpdFile.ValidationErrors {
					assert.Contains(t, errL[i].Error(), fpdFile.ValidationErrors[i], "Incorrect first party data warning message")
				}
			}

		}
	}
}

func TestExtractFPDForBidders(t *testing.T) {

	if specFiles, err := ioutil.ReadDir("./tests/extractfpdforbidders"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./tests/extractfpdforbidders/" + specFile.Name()
			fpdFile, err := loadFpdFile(fileName)

			if err != nil {
				t.Errorf("Unable to load file: %s", fileName)
			}

			var resRequest openrtb2.BidRequest
			err = json.Unmarshal(fpdFile.OutputRequestData, &resRequest)
			if err != nil {
				t.Errorf("Unable to unmarshal input request: %s", fileName)
			}

			req := &openrtb_ext.RequestWrapper{}
			req.BidRequest = &openrtb2.BidRequest{}
			err = json.Unmarshal(fpdFile.InputRequestData, req.BidRequest)
			assert.NoError(t, err, "Error should be nil")

			resultFPD, errL := ExtractFPDForBidders(req)

			if len(fpdFile.ValidationErrors) > 0 {
				assert.Equal(t, len(fpdFile.ValidationErrors), len(errL), "")
				//errors can be returned in a different order from how they are specified in file
				for _, actualValidationErr := range errL {
					errorContainsText := false
					for _, expectedValidationErr := range fpdFile.ValidationErrors {
						if strings.Contains(actualValidationErr.Error(), expectedValidationErr) {
							errorContainsText = true
							break
						}
					}
					assert.True(t, errorContainsText, "Incorrect validation message")
				}
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

			if resRequest.Site != nil {
				if len(resRequest.Site.Ext) > 0 {
					assert.JSONEq(t, string(resRequest.Site.Ext), string(req.BidRequest.Site.Ext), "Incorrect site in request")
					resRequest.Site.Ext = nil
					req.BidRequest.Site.Ext = nil
				}
				assert.Equal(t, resRequest.Site, req.BidRequest.Site, "Incorrect site in request")
			}
			if resRequest.App != nil {
				if len(resRequest.App.Ext) > 0 {
					assert.JSONEq(t, string(resRequest.App.Ext), string(req.BidRequest.App.Ext), "Incorrect app in request")
					resRequest.App.Ext = nil
					req.BidRequest.App.Ext = nil
				}
				assert.Equal(t, resRequest.App, req.BidRequest.App, "Incorrect app in request")
			}
			if resRequest.User != nil {
				if len(resRequest.User.Ext) > 0 {
					assert.JSONEq(t, string(resRequest.User.Ext), string(req.BidRequest.User.Ext), "Incorrect user in request")
					resRequest.User.Ext = nil
					req.BidRequest.User.Ext = nil
				}
				assert.Equal(t, resRequest.User, req.BidRequest.User, "Incorrect user in request")
			}

		}
	}
}

func loadFpdFile(filename string) (fpdFile, error) {
	var fileData fpdFile
	fileContents, err := ioutil.ReadFile(filename)
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
	BiddersFPD         map[openrtb_ext.BidderName]*openrtb_ext.ORTB2      `json:"biddersFPD,omitempty"`
	BiddersFPDResolved map[openrtb_ext.BidderName]*ResolvedFirstPartyData `json:"biddersFPDResolved,omitempty"`
	FirstPartyData     map[string]json.RawMessage                         `json:"firstPartyData,omitempty"`
	ValidationErrors   []string                                           `json:"validationErrors,omitempty"`
}
