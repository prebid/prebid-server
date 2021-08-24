package firstpartydata

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/jsonutil"
)

const (
	site = "site"
	app  = "app"
	user = "user"
	data = "data"
	ext  = "ext"
)

func GetGlobalFPDData(request []byte) ([]byte, map[string][]byte, error) {
	//If {site,app,user}.ext.data exists, collect it and remove {site,app,user}.ext.data

	fpdReqData := make(map[string][]byte, 0)
	request, siteFPD, err := jsonutil.FindAndDropElement(request, site, ext, data)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[site] = siteFPD

	request, appFPD, err := jsonutil.FindAndDropElement(request, app, ext, data)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[app] = appFPD

	fpdReqData[user] = []byte{}
	request, userFPD, err := jsonutil.FindAndDropElement(request, user, ext, data)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[user] = userFPD

	return request, fpdReqData, nil
}

func BuildFPD(bidRequest *openrtb2.BidRequest, fpdBidderData map[openrtb_ext.BidderName]*openrtb_ext.FPDData, globalFPD map[string][]byte) (map[openrtb_ext.BidderName]*openrtb_ext.FPDData, error) {

	// If an attribute doesn't pass defined validation checks,
	// entire request should be rejected with error message

	resolvedFpdData := make(map[openrtb_ext.BidderName]*openrtb_ext.FPDData)

	for bidderName, fpdConfig := range fpdBidderData {

		resolvedFpdConfig := &openrtb_ext.FPDData{}

		newUser, err := resolveUser(fpdConfig.User, bidRequest.User, globalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.User = newUser

		newApp, err := resolveApp(fpdConfig.App, bidRequest.App, globalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.App = newApp

		newSite, err := resolveSite(fpdConfig.Site, bidRequest.Site, globalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.Site = newSite

		resolvedFpdData[bidderName] = resolvedFpdConfig
	}
	return resolvedFpdData, nil
}

func resolveUser(fpdConfigUser *openrtb2.User, bidRequestUser *openrtb2.User, globalFPD map[string][]byte) (*openrtb2.User, error) {
	if fpdConfigUser != nil {
		if bidRequestUser == nil {
			return fpdConfigUser, nil
		} else {
			resUser, err := mergeFPD(bidRequestUser, fpdConfigUser, globalFPD, user)
			if err != nil {
				return nil, err
			}
			newUser := &openrtb2.User{}
			err = json.Unmarshal(resUser, newUser)
			if err != nil {
				return nil, err
			}
			return newUser, err
		}
	}
	return nil, nil
}

func resolveSite(fpdConfigSite *openrtb2.Site, bidRequestSite *openrtb2.Site, globalFPD map[string][]byte) (*openrtb2.Site, error) {
	if fpdConfigSite != nil {
		if bidRequestSite == nil {
			return fpdConfigSite, nil
		} else {
			resSite, err := mergeFPD(bidRequestSite, fpdConfigSite, globalFPD, site)
			if err != nil {
				return nil, err
			}
			newSite := &openrtb2.Site{}
			err = json.Unmarshal(resSite, newSite)
			if err != nil {
				return nil, err
			}
			return newSite, nil
		}
	}
	return nil, nil
}

func resolveApp(fpdConfigApp *openrtb2.App, bidRequestApp *openrtb2.App, globalFPD map[string][]byte) (*openrtb2.App, error) {
	if fpdConfigApp != nil {
		if bidRequestApp == nil {
			return fpdConfigApp, nil
		} else {
			resApp, err := mergeFPD(bidRequestApp, fpdConfigApp, globalFPD, app)
			if err != nil {
				return nil, err
			}

			newApp := &openrtb2.App{}
			err = json.Unmarshal(resApp, newApp)
			if err != nil {
				return nil, err
			}
			return newApp, nil
		}
	}
	return nil, nil
}

func mergeFPD(input interface{}, fpd interface{}, data map[string][]byte, value string) ([]byte, error) {

	inputByte, err := json.Marshal(input)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Invalid first party data input: %s", input),
		}
	}
	fpdByte, err := json.Marshal(fpd)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Invalid first party data: %s", fpd),
		}
	}
	resultMerged, err := jsonpatch.MergePatch(inputByte, fpdByte)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Unable to merge first party data for: %s", value),
		}
	}
	//merge global fpd to final app/site/user.ext.data
	if len(data[value]) > 0 {
		extData := buildExtData(data[value])
		resultMerged, err = jsonpatch.MergePatch(resultMerged, extData)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprint("Unable to merge global fpd to final app/site/user.ext.data"),
			}
		}
	}

	return resultMerged, err
}

func buildExtData(data []byte) []byte {
	res := []byte(`{"ext":{"data":`)
	res = append(res, data...)
	res = append(res, []byte(`}}`)...)
	return res
}

func PreprocessBidderFPD(reqExtPrebid openrtb_ext.ExtRequestPrebid) (map[openrtb_ext.BidderName]*openrtb_ext.FPDData, openrtb_ext.ExtRequestPrebid) {
	//map to store bidder configs to process
	fpdData := make(map[openrtb_ext.BidderName]*openrtb_ext.FPDData)

	if reqExtPrebid.Data != nil && len(reqExtPrebid.Data.Bidders) != 0 && reqExtPrebid.BidderConfigs != nil {

		//every entry in ext.prebid.bidderconfig[].bidders would also need to be in ext.prebid.data.bidders or it will be ignored
		bidderTable := make(map[string]interface{}) //just need to check existence of the element in map
		for _, bidder := range reqExtPrebid.Data.Bidders {
			bidderTable[bidder] = true
		}

		for _, bidderConfig := range *reqExtPrebid.BidderConfigs {
			for _, bidder := range bidderConfig.Bidders {
				if _, present := bidderTable[bidder]; present {

					if fpdData[openrtb_ext.BidderName(bidder)] == nil {
						fpdData[openrtb_ext.BidderName(bidder)] = bidderConfig.FPDConfig.FPDData
					} else {
						//this will overwrite previously set site/app/user.
						//Last defined bidder-specific config will take precedence
						fpdBidderData := fpdData[openrtb_ext.BidderName(bidder)]
						if bidderConfig.FPDConfig != nil && bidderConfig.FPDConfig.FPDData != nil {
							if bidderConfig.FPDConfig.FPDData.Site != nil {
								fpdBidderData.Site = bidderConfig.FPDConfig.FPDData.Site
							}
							if bidderConfig.FPDConfig.FPDData.App != nil {
								fpdBidderData.App = bidderConfig.FPDConfig.FPDData.App
							}
							if bidderConfig.FPDConfig.FPDData.User != nil {
								fpdBidderData.User = bidderConfig.FPDConfig.FPDData.User
							}
						}
					}
				}
			}
		}
	}

	reqExtPrebid.BidderConfigs = nil
	if reqExtPrebid.Data != nil {
		reqExtPrebid.Data.Bidders = nil
	}

	return fpdData, reqExtPrebid
}

func ValidateFPDConfig(reqExtPrebid openrtb_ext.ExtRequestPrebid) error {

	//Both FPD global and bidder specific permissions are specified
	if reqExtPrebid.Data == nil && reqExtPrebid.BidderConfigs == nil {
		return nil
	}

	if reqExtPrebid.Data != nil && len(reqExtPrebid.Data.Bidders) != 0 && reqExtPrebid.BidderConfigs == nil {
		return errors.New(`request.ext.prebid.data.bidders are specified but reqExtPrebid.BidderConfigs are not`)
	}
	if reqExtPrebid.Data != nil && len(reqExtPrebid.Data.Bidders) == 0 && reqExtPrebid.BidderConfigs != nil {
		return errors.New(`request.ext.prebid.data.bidders are not specified but reqExtPrebid.BidderConfigs are`)
	}

	if reqExtPrebid.Data == nil && reqExtPrebid.BidderConfigs != nil {
		return errors.New(`request.ext.prebid.data is not specified but reqExtPrebid.BidderConfigs are`)
	}

	return nil
}
