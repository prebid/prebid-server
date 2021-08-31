package firstpartydata

import (
	"encoding/json"
	"fmt"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/jsonutil"
)

const (
	siteKey = "site"
	appKey  = "app"
	userKey = "user"
	dataKey = "data"
	extKey  = "ext"

	userDataKey        = "userData"
	appContentDataKey  = "appContentData"
	siteContentDataKey = "siteContentData"
)

func GetGlobalFPDData(request []byte) ([]byte, map[string][]byte, error) {
	//If {site,app,user}.ext.data exists, collect it and remove {site,app,user}.ext.data

	fpdReqData := make(map[string][]byte, 3)
	request, siteFPD, err := jsonutil.FindAndDropElement(request, siteKey, extKey, dataKey)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[siteKey] = siteFPD

	request, appFPD, err := jsonutil.FindAndDropElement(request, appKey, extKey, dataKey)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[appKey] = appFPD

	request, userFPD, err := jsonutil.FindAndDropElement(request, userKey, extKey, dataKey)
	if err != nil {
		return request, nil, err
	}
	fpdReqData[userKey] = userFPD

	return request, fpdReqData, nil
}

func ExtractOpenRtbGlobalFPD(bidRequest *openrtb2.BidRequest) map[string][]openrtb2.Data {
	//Delete user.data and {app/site}.content.data from request

	openRtbGlobalFPD := make(map[string][]openrtb2.Data, 0)
	if bidRequest.User != nil && len(bidRequest.User.Data) > 0 {
		openRtbGlobalFPD[userDataKey] = bidRequest.User.Data
		bidRequest.User.Data = nil
	}

	if bidRequest.Site != nil && bidRequest.Site.Content != nil && len(bidRequest.Site.Content.Data) > 0 {
		openRtbGlobalFPD[siteContentDataKey] = bidRequest.Site.Content.Data
		bidRequest.Site.Content.Data = nil
	}

	if bidRequest.App != nil && bidRequest.App.Content != nil && len(bidRequest.App.Content.Data) > 0 {
		openRtbGlobalFPD[appContentDataKey] = bidRequest.App.Content.Data
		bidRequest.App.Content.Data = nil
	}

	return openRtbGlobalFPD

}

func BuildResolvedFPDForBidders(bidRequest *openrtb2.BidRequest, fpdBidderData map[openrtb_ext.BidderName]*openrtb_ext.FPDData, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data) (map[openrtb_ext.BidderName]*openrtb_ext.FPDData, error) {

	// If an attribute doesn't pass defined validation checks,
	// entire request should be rejected with error message

	resolvedFpdData := make(map[openrtb_ext.BidderName]*openrtb_ext.FPDData)

	for bidderName, fpdConfig := range fpdBidderData {

		resolvedFpdConfig := &openrtb_ext.FPDData{}

		newUser, err := resolveUser(fpdConfig.User, bidRequest.User, globalFPD, openRtbGlobalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.User = newUser

		newApp, err := resolveApp(fpdConfig.App, bidRequest.App, globalFPD, openRtbGlobalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.App = newApp

		newSite, err := resolveSite(fpdConfig.Site, bidRequest.Site, globalFPD, openRtbGlobalFPD)
		if err != nil {
			return nil, err
		}
		resolvedFpdConfig.Site = newSite

		resolvedFpdData[bidderName] = resolvedFpdConfig
	}
	return resolvedFpdData, nil
}

func resolveUser(fpdConfigUser *openrtb2.User, bidRequestUser *openrtb2.User, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data) (*openrtb2.User, error) {
	if bidRequestUser == nil && fpdConfigUser == nil {
		return nil, nil
	}
	if bidRequestUser == nil {
		bidRequestUser = &openrtb2.User{}
	}
	if fpdConfigUser == nil {
		fpdConfigUser = &openrtb2.User{}
	}

	resUser, err := mergeFPD(bidRequestUser, fpdConfigUser, globalFPD, userKey)
	if err != nil {
		return nil, err
	}

	newUser := &openrtb2.User{}
	err = json.Unmarshal(resUser, newUser)
	if err != nil {
		return nil, err
	}
	if len(openRtbGlobalFPD[userDataKey]) > 0 {
		newUser.Data = openRtbGlobalFPD[userDataKey]
	}
	return newUser, nil

}

func resolveSite(fpdConfigSite *openrtb2.Site, bidRequestSite *openrtb2.Site, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data) (*openrtb2.Site, error) {

	if bidRequestSite == nil && fpdConfigSite == nil {
		return nil, nil
	}
	if bidRequestSite == nil {
		bidRequestSite = &openrtb2.Site{}
	}
	if fpdConfigSite == nil {
		fpdConfigSite = &openrtb2.Site{}
	}

	resSite, err := mergeFPD(bidRequestSite, fpdConfigSite, globalFPD, siteKey)
	if err != nil {
		return nil, err
	}

	newSite := &openrtb2.Site{}
	err = json.Unmarshal(resSite, newSite)
	if err != nil {
		return nil, err
	}
	if len(openRtbGlobalFPD[siteContentDataKey]) > 0 {
		if newSite.Content != nil {
			newSite.Content.Data = openRtbGlobalFPD[siteContentDataKey]
		} else {
			newSiteContent := &openrtb2.Content{Data: openRtbGlobalFPD[siteContentDataKey]}
			newSite.Content = newSiteContent
		}
	}
	return newSite, nil
}

func resolveApp(fpdConfigApp *openrtb2.App, bidRequestApp *openrtb2.App, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data) (*openrtb2.App, error) {

	if bidRequestApp == nil && fpdConfigApp == nil {
		return nil, nil
	}
	if bidRequestApp == nil {
		bidRequestApp = &openrtb2.App{}
	}
	if fpdConfigApp == nil {
		fpdConfigApp = &openrtb2.App{}
	}

	resApp, err := mergeFPD(bidRequestApp, fpdConfigApp, globalFPD, appKey)
	if err != nil {
		return nil, err
	}

	newApp := &openrtb2.App{}
	err = json.Unmarshal(resApp, newApp)
	if err != nil {
		return nil, err
	}

	if len(openRtbGlobalFPD[appContentDataKey]) > 0 {
		if newApp.Content != nil {
			newApp.Content.Data = openRtbGlobalFPD[appContentDataKey]
		} else {
			newAppContent := &openrtb2.Content{Data: openRtbGlobalFPD[appContentDataKey]}
			newApp.Content = newAppContent
		}
	}
	return newApp, nil
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

		//every bidder in ext.prebid.data.bidders should receive fpd data if defined
		bidderTable := make(map[string]interface{}) //just need to check existence of the element in map
		for _, bidder := range reqExtPrebid.Data.Bidders {
			bidderTable[bidder] = true
			fpdData[openrtb_ext.BidderName(bidder)] = &openrtb_ext.FPDData{}
		}

		for _, bidderConfig := range *reqExtPrebid.BidderConfigs {
			for _, bidder := range bidderConfig.Bidders {
				if _, present := bidderTable[bidder]; present {
					//this will overwrite previously set site/app/user.
					//Last defined bidder-specific config will take precedence
					fpdBidderData := fpdData[openrtb_ext.BidderName(bidder)]
					if bidderConfig.Config != nil && bidderConfig.Config.FPDData != nil {
						if bidderConfig.Config.FPDData.Site != nil {
							fpdBidderData.Site = bidderConfig.Config.FPDData.Site
						}
						if bidderConfig.Config.FPDData.App != nil {
							fpdBidderData.App = bidderConfig.Config.FPDData.App
						}
						if bidderConfig.Config.FPDData.User != nil {
							fpdBidderData.User = bidderConfig.Config.FPDData.User
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
