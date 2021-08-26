package firstpartydata

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/jsonutil"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"testing"
)

func TestGetFPDData(t *testing.T) {

	testCases := []struct {
		description     string
		input           []byte
		output          []byte
		expectedFpdData map[string][]byte
		errorExpected   bool
		errorContains   string
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
				   "data": {"somesitefpd": "sitefpdDataTest"}
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
				  "ext": {}
  				},
  				"user": {
  				  "id": "reqUserID",
  				  "yob": 1982,
  				  "gender": "M",
				  "ext": {}
  				},
  				"app": {
  				  "id": "appId",
  				  "data": 123,
				  "ext": {}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			expectedFpdData: map[string][]byte{
				"site": []byte(`{"somesitefpd": "sitefpdDataTest"}`),
				"user": []byte(`{"someuserfpd": "userfpdDataTest"}`),
				"app":  []byte(`{"someappfpd": "appfpdDataTest"}`),
			},
			errorExpected: false,
			errorContains: "",
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
			expectedFpdData: map[string][]byte{
				"app":  []byte(`{"someappfpd": "appfpdDataTest"}`),
				"user": {},
				"site": {},
			},
			errorExpected: false,
			errorContains: "",
		}, {
			description: "Site FPD different format",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
				  "ext": {
					"data": {"someappfpd": true}
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
                  "ext": {}
  				},
  				"app": {
  				  "id": "appId"
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			expectedFpdData: map[string][]byte{
				"app":  {},
				"user": {},
				"site": []byte(`{"someappfpd": true}`),
			},
			errorExpected: false,
			errorContains: "",
		},
		{
			description: "Malformed input",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "ext": {"data": meappfpd": "appfpdDataTest"}}
  				},
  				: 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439
  				}
			}`),
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "ext": {"data": meappfpd": "appfpdDataTest"}}
  				},
  				: 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439
  				}
			}`),
			expectedFpdData: map[string][]byte{},
			errorExpected:   true,
			errorContains:   "Unknown value type",
		},
	}
	for _, test := range testCases {
		res, fpd, err := GetGlobalFPDData(test.input)

		if test.errorExpected {
			assert.Error(t, err, "Error should not be nil")
			//result should be still returned
			assert.Equal(t, string(test.output), string(res), "Result is incorrect")
			assert.True(t, strings.Contains(err.Error(), test.errorContains))
		} else {
			assert.NoError(t, err, "Error should be nil")
			assert.JSONEq(t, string(test.output), string(res), "Result is incorrect")
			assert.Equal(t, test.expectedFpdData, fpd, "FPD is incorrect")
		}

	}
}

func TestPreprocessFPD(t *testing.T) {

	if specFiles, err := ioutil.ReadDir("./tests/preprocessfpd"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./tests/preprocessfpd/" + specFile.Name()

			fpdFile, err := loadFpdFile(fileName)
			if err != nil {
				t.Errorf("Unable to load file: %s", fileName)
			}
			var extReq openrtb_ext.ExtRequestPrebid
			err = json.Unmarshal(fpdFile.InputRequestData, &extReq)
			if err != nil {
				t.Errorf("Unable to unmarshal input request: %s", fileName)
			}
			fpdData, reqExtPrebid := PreprocessBidderFPD(extReq)

			if reqExtPrebid.Data != nil {
				assert.Nil(t, reqExtPrebid.Data.Bidders, "Global FPD config should be removed from request")
			}
			assert.Nil(t, reqExtPrebid.BidderConfigs, "Bidder specific FPD config should be removed from request")

			assert.Equal(t, len(fpdFile.BiddersFPD), len(fpdData), "Incorrect fpd data")

			for k, v := range fpdFile.BiddersFPD {

				if v.Site != nil {
					tempSiteExt := fpdData[k].Site.Ext
					jsonutil.DiffJson(t, "site.ext is incorrect", v.Site.Ext, tempSiteExt)
					//compare extensions first and the site objects without extensions
					//in case two or more bidders share same config(pointer), ext should be returned back
					v.Site.Ext = nil
					fpdData[k].Site.Ext = nil
					assert.Equal(t, v.Site, fpdData[k].Site, "Incorrect site fpd data")
					fpdData[k].Site.Ext = tempSiteExt
				}

				if v.App != nil {

					tempAppExt := fpdData[k].App.Ext
					jsonutil.DiffJson(t, "app.ext is incorrect", v.App.Ext, tempAppExt)
					//compare extensions first and the app objects without extensions
					v.App.Ext = nil
					fpdData[k].App.Ext = nil
					assert.Equal(t, v.App, fpdData[k].App, "Incorrect app fpd data")
					fpdData[k].App.Ext = tempAppExt
				}

				if v.User != nil {
					tempUserExt := fpdData[k].User.Ext
					jsonutil.DiffJson(t, "user.ext is incorrect", v.User.Ext, tempUserExt)
					//compare extensions first and the user objects without extensions
					v.User.Ext = nil
					fpdData[k].User.Ext = nil
					assert.Equal(t, v.User, fpdData[k].User, "Incorrect user fpd data")
					fpdData[k].User.Ext = tempUserExt
				}

			}
		}
	}
}

func TestApplyFPD(t *testing.T) {

	if specFiles, err := ioutil.ReadDir("./tests/applyfpd"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./tests/applyfpd/" + specFile.Name()

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

			reqFPD := make(map[string][]byte, 0)

			reqFPD["site"] = fpdFile.FirstPartyData["site"]
			reqFPD["app"] = fpdFile.FirstPartyData["app"]
			reqFPD["user"] = fpdFile.FirstPartyData["user"]

			resultFPD, err := BuildResolvedFPDForBidders(&inputReq, fpdFile.BiddersFPD, reqFPD, nil)

			assert.NoError(t, err, "No errors should be returned")
			assert.Equal(t, inputReq, inputReqCopy, "Original request should not be modified")

			biddrFPD := resultFPD["appnexus"]

			if biddrFPD.Site != nil && len(biddrFPD.Site.Ext) > 0 {
				resSiteExt := biddrFPD.Site.Ext
				expectedSiteExt := outputReq.Site.Ext
				biddrFPD.Site.Ext = nil
				outputReq.Site.Ext = nil
				jsonutil.DiffJson(t, "site.ext is incorrect", resSiteExt, expectedSiteExt)
			}
			if biddrFPD.App != nil && len(biddrFPD.App.Ext) > 0 {
				resAppExt := biddrFPD.App.Ext
				expectedAppExt := outputReq.App.Ext
				biddrFPD.App.Ext = nil
				outputReq.App.Ext = nil
				jsonutil.DiffJson(t, "app.ext is incorrect", resAppExt, expectedAppExt)
			}
			if biddrFPD.User != nil && len(biddrFPD.User.Ext) > 0 {
				resUserExt := biddrFPD.User.Ext
				expectedUserExt := outputReq.User.Ext
				biddrFPD.User.Ext = nil
				outputReq.User.Ext = nil
				jsonutil.DiffJson(t, "user.ext is incorrect", resUserExt, expectedUserExt)
			}
		}
	}
}

func TestMergeFPDData(t *testing.T) {

	if specFiles, err := ioutil.ReadDir("./tests/mergefpd"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./tests/mergefpd/" + specFile.Name()

			fpdFile, err := loadFpdFile(fileName)
			if err != nil {
				t.Errorf("Unable to load file: %s", fileName)
			}
			rawData := []byte(fpdFile.FirstPartyData["site"])
			firstPartyData := make(map[string][]byte)
			firstPartyData["site"] = rawData

			fpdData := fpdFile.BiddersFPD["appnexus"].Site

			resSite, err := mergeFPD(fpdFile.InputRequestData, fpdData, firstPartyData, "site")

			assert.Nil(t, err, "Error should be nil")

			jsonutil.DiffJson(t, "Result is incorrect"+fileName, resSite, fpdFile.OutputRequestData)

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
	InputRequestData  json.RawMessage                                 `json:"inputRequestData,omitempty"`
	OutputRequestData json.RawMessage                                 `json:"outputRequestData,omitempty"`
	BiddersFPD        map[openrtb_ext.BidderName]*openrtb_ext.FPDData `json:"biddersFPD,omitempty"`
	FirstPartyData    map[string]json.RawMessage                      `json:"firstPartyData,omitempty"`
}

func TestValidateFPDConfig(t *testing.T) {

	bidderConfigs := &[]openrtb_ext.BidderConfig{
		{
			Bidders: []string{"testBidder1"},
			Config: &openrtb_ext.Config{
				FPDData: &openrtb_ext.FPDData{
					Site: &openrtb2.Site{ID: "testBidder1SiteId"},
				},
			},
		},
	}

	bidderConfigsNoConfigs := &[]openrtb_ext.BidderConfig{
		{
			Bidders: []string{"testBidder1"},
			Config:  nil,
		},
	}

	testCases := []struct {
		description   string
		reqExtPrebid  openrtb_ext.ExtRequestPrebid
		errorExpected bool
		errorContains string
	}{
		{
			description: "Valid config both present",
			reqExtPrebid: openrtb_ext.ExtRequestPrebid{
				Data: &openrtb_ext.ExtRequestPrebidData{
					Bidders: []string{"testBidder1"},
				},
				BidderConfigs: bidderConfigs,
			},
			errorExpected: false,
			errorContains: "",
		},
		{
			description: "Valid config both not present",
			reqExtPrebid: openrtb_ext.ExtRequestPrebid{
				Data:          nil,
				BidderConfigs: nil,
			},
			errorExpected: false,
			errorContains: "",
		},
		{
			description: "Invalid config data nil",
			reqExtPrebid: openrtb_ext.ExtRequestPrebid{
				Data:          nil,
				BidderConfigs: bidderConfigs,
			},
			errorExpected: true,
			errorContains: "request.ext.prebid.data is not specified but reqExtPrebid.BidderConfigs are",
		},
		{
			description: "Invalid config no bidders",
			reqExtPrebid: openrtb_ext.ExtRequestPrebid{
				Data: &openrtb_ext.ExtRequestPrebidData{
					Bidders: []string{"testBidder1"},
				},
				BidderConfigs: nil,
			},
			errorExpected: true,
			errorContains: "request.ext.prebid.data.bidders are specified but reqExtPrebid.BidderConfigs are",
		},
		{
			description: "Invalid config no configs",
			reqExtPrebid: openrtb_ext.ExtRequestPrebid{
				Data: &openrtb_ext.ExtRequestPrebidData{
					Bidders: []string{},
				},
				BidderConfigs: bidderConfigsNoConfigs,
			},
			errorExpected: true,
			errorContains: "request.ext.prebid.data.bidders are not specified but reqExtPrebid.BidderConfigs are",
		},
	}
	for _, test := range testCases {
		err := ValidateFPDConfig(test.reqExtPrebid)

		if test.errorExpected {
			assert.NotNil(t, err, "error expected")
			assert.True(t, strings.Contains(err.Error(), test.errorContains))
		} else {
			assert.Nil(t, err, "error is not expected")
		}
	}
}
