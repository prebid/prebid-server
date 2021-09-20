package firstpartydata

import (
	"fmt"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/jsonutil"
)

const (
	siteKey = "site"
	appKey  = "app"
	userKey = "user"
	dataKey = "data"

	userDataKey        = "userData"
	appContentDataKey  = "appContentData"
	siteContentDataKey = "siteContentData"
)

func ExtractGlobalFPDData(req *openrtb_ext.RequestWrapper) (map[string][]byte, error) {
	//If {site,app,user}.ext.data exists, collect it and remove {site,app,user}.ext.data

	fpdReqData := make(map[string][]byte, 3)

	siteExt, err := req.GetSiteExt()
	if err != nil {
		return nil, err
	}

	if siteExt != nil && len(siteExt.GetExt()[dataKey]) > 0 {
		//set global site fpd
		fpdReqData[siteKey] = siteExt.GetExt()[dataKey]
		//remove req.Site.Ext.data from request
		newSiteExt, err := jsonutil.DropElement(req.Site.Ext, dataKey)
		if err != nil {
			return nil, err
		}
		req.Site.Ext = newSiteExt
	}

	appExt, err := req.GetAppExt()
	if err != nil {
		return nil, err
	}
	if appExt != nil && len(appExt.GetExt()[dataKey]) > 0 {
		//set global app fpd
		fpdReqData[appKey] = appExt.GetExt()[dataKey]
		//remove req.App.Ext.data from request
		newAppExt, err := jsonutil.DropElement(req.App.Ext, dataKey)
		if err != nil {
			return nil, err
		}
		req.App.Ext = newAppExt
	}

	userExt, err := req.GetUserExt()
	if err != nil {
		return nil, err
	}
	if req.User != nil && len(userExt.GetExt()[dataKey]) > 0 {
		//set global user fpd
		fpdReqData[userKey] = userExt.GetExt()[dataKey]
		//remove req.App.Ext.data from request
		newUserExt, err := jsonutil.DropElement(req.User.Ext, dataKey)
		if err != nil {
			return nil, err
		}
		req.User.Ext = newUserExt
	}

	return fpdReqData, nil
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

func ResolveFPDData(bidRequest *openrtb2.BidRequest, fpdBidderConfigData map[openrtb_ext.BidderName]*openrtb_ext.ORTB2, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, biddersWithGlobalFPD []string) (map[openrtb_ext.BidderName]*openrtb_ext.ORTB2, []error) {
	errL := []error{}
	// If an attribute doesn't pass defined validation checks,
	// entire request should be rejected with error message

	resolvedFpdData := make(map[openrtb_ext.BidderName]*openrtb_ext.ORTB2)

	//convert list to map to optimize check if value exists
	globalBiddersTable := make(map[string]struct{}) //just need to check existence of the element in map
	for _, bidderName := range biddersWithGlobalFPD {
		globalBiddersTable[bidderName] = struct{}{}
	}

	for bName, fpdConfig := range fpdBidderConfigData {
		bidderName := string(bName)

		_, hasGlobalFPD := globalBiddersTable[bidderName]

		resolvedFpdConfig := &openrtb_ext.ORTB2{}

		newUser, err := resolveUser(fpdConfig.User, bidRequest.User, globalFPD, openRtbGlobalFPD, hasGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.User = newUser

		newApp, err := resolveApp(fpdConfig.App, bidRequest.App, globalFPD, openRtbGlobalFPD, hasGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.App = newApp

		newSite, err := resolveSite(fpdConfig.Site, bidRequest.Site, globalFPD, openRtbGlobalFPD, hasGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		if len(errL) == 0 {
			resolvedFpdConfig.Site = newSite

			resolvedFpdData[bName] = resolvedFpdConfig
		}
	}
	return resolvedFpdData, errL
}

func resolveUser(fpdConfigUser *openrtb2.User, bidRequestUser *openrtb2.User, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool, bidderName string) (*openrtb2.User, error) {

	if bidRequestUser == nil && fpdConfigUser == nil {
		return nil, nil
	}

	if bidRequestUser == nil && fpdConfigUser != nil {
		return nil, fmt.Errorf("incorrect First Party Data for bidder %s: User object is not defined in request, but defined in FPD config", bidderName)
	}

	newUser := *bidRequestUser
	var err error

	if hasGlobalFPD {
		//apply global fpd
		if len(globalFPD[userKey]) > 0 {
			extData := buildExtData(globalFPD[userKey])
			if len(newUser.Ext) > 0 {
				newUser.Ext, err = jsonpatch.MergePatch(newUser.Ext, extData)
			} else {
				newUser.Ext = extData
			}
		}
		if len(openRtbGlobalFPD[userDataKey]) > 0 {
			newUser.Data = openRtbGlobalFPD[userDataKey]
		}
	}
	if fpdConfigUser != nil {
		//apply bidder specific fpd if present
		newUser, err = mergeUsers(&newUser, fpdConfigUser)
	}

	return &newUser, err
}

func mergeUsers(original *openrtb2.User, fpdConfigUser *openrtb2.User) (openrtb2.User, error) {

	var err error
	newUser := openrtb2.User{}
	newUser = *original
	newUser.Keywords = fpdConfigUser.Keywords
	newUser.Gender = fpdConfigUser.Gender
	newUser.Yob = fpdConfigUser.Yob

	if len(fpdConfigUser.Ext) > 0 {
		if len(original.Ext) > 0 {
			newUser.Ext, err = jsonpatch.MergePatch(original.Ext, fpdConfigUser.Ext)
		} else {
			newUser.Ext = fpdConfigUser.Ext
		}
	}

	return newUser, err
}

func resolveSite(fpdConfigSite *openrtb2.Site, bidRequestSite *openrtb2.Site, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool, bidderName string) (*openrtb2.Site, error) {

	if bidRequestSite == nil && fpdConfigSite == nil {
		return nil, nil
	}
	if bidRequestSite == nil && fpdConfigSite != nil {
		return nil, fmt.Errorf("incorrect First Party Data for bidder %s: Site object is not defined in request, but defined in FPD config", bidderName)
	}

	newSite := *bidRequestSite
	var err error

	if hasGlobalFPD {
		//apply global fpd
		if len(globalFPD[siteKey]) > 0 {
			extData := buildExtData(globalFPD[siteKey])
			if len(newSite.Ext) > 0 {
				newSite.Ext, err = jsonpatch.MergePatch(newSite.Ext, extData)
			} else {
				newSite.Ext = extData
			}
		}
		if len(openRtbGlobalFPD[siteContentDataKey]) > 0 {
			if newSite.Content != nil {
				contentCopy := *newSite.Content
				contentCopy.Data = openRtbGlobalFPD[siteContentDataKey]
				newSite.Content = &contentCopy
			} else {
				newSiteContent := &openrtb2.Content{Data: openRtbGlobalFPD[siteContentDataKey]}
				newSite.Content = newSiteContent
			}
		}
	}

	if fpdConfigSite != nil {
		//apply bidder specific fpd if present
		//result site should have ID or Page, fpd becomes incorrect if it overwrites page to empty one and ID is empty in original site
		if fpdConfigSite.Page == "" && newSite.Page != "" && newSite.ID == "" {
			return nil, fmt.Errorf("incorrect First Party Data for bidder %s: Site object cannot set empty page if req.site.id is empty", bidderName)

		}
		newSite, err = mergeSites(&newSite, fpdConfigSite)
	}
	return &newSite, err

}

func mergeSites(originalSite *openrtb2.Site, fpdConfigSite *openrtb2.Site) (openrtb2.Site, error) {

	var err error
	newSite := openrtb2.Site{}
	newSite = *originalSite

	newSite.Name = fpdConfigSite.Name
	newSite.Domain = fpdConfigSite.Domain
	newSite.Cat = fpdConfigSite.Cat
	newSite.SectionCat = fpdConfigSite.SectionCat
	newSite.PageCat = fpdConfigSite.PageCat
	newSite.Page = fpdConfigSite.Page
	newSite.Search = fpdConfigSite.Search
	newSite.Keywords = fpdConfigSite.Keywords

	if len(fpdConfigSite.Ext) > 0 {
		if len(originalSite.Ext) > 0 {
			newSite.Ext, err = jsonpatch.MergePatch(originalSite.Ext, fpdConfigSite.Ext)
		} else {
			newSite.Ext = fpdConfigSite.Ext
		}
	}

	return newSite, err
}

func resolveApp(fpdConfigApp *openrtb2.App, bidRequestApp *openrtb2.App, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool, bidderName string) (*openrtb2.App, error) {

	if bidRequestApp == nil && fpdConfigApp == nil {
		return nil, nil
	}

	if bidRequestApp == nil && fpdConfigApp != nil {
		return nil, fmt.Errorf("incorrect First Party Data for bidder %s: App object is not defined in request, but defined in FPD config", bidderName)
	}

	newApp := *bidRequestApp
	var err error

	if hasGlobalFPD {
		//apply global fpd if exists
		if len(globalFPD[appKey]) > 0 {
			extData := buildExtData(globalFPD[appKey])
			if len(newApp.Ext) > 0 {
				newApp.Ext, err = jsonpatch.MergePatch(newApp.Ext, extData)
			} else {
				newApp.Ext = extData
			}
		}
		if len(openRtbGlobalFPD[appContentDataKey]) > 0 {
			if newApp.Content != nil {
				contentCopy := *newApp.Content
				contentCopy.Data = openRtbGlobalFPD[appContentDataKey]
				newApp.Content = &contentCopy
			} else {
				newAppContent := &openrtb2.Content{Data: openRtbGlobalFPD[appContentDataKey]}
				newApp.Content = newAppContent
			}
		}
	}

	if fpdConfigApp != nil {
		//apply bidder specific fpd if present
		newApp, err = mergeApps(&newApp, fpdConfigApp)
	}

	return &newApp, err
}

func mergeApps(originalApp *openrtb2.App, fpdConfigApp *openrtb2.App) (openrtb2.App, error) {

	var err error
	newApp := openrtb2.App{}
	newApp = *originalApp

	newApp.Name = fpdConfigApp.Name
	newApp.Bundle = fpdConfigApp.Bundle
	newApp.Domain = fpdConfigApp.Domain
	newApp.StoreURL = fpdConfigApp.StoreURL
	newApp.Cat = fpdConfigApp.Cat
	newApp.SectionCat = fpdConfigApp.SectionCat
	newApp.PageCat = fpdConfigApp.PageCat
	newApp.Ver = fpdConfigApp.Ver
	newApp.Keywords = fpdConfigApp.Keywords

	if len(fpdConfigApp.Ext) > 0 {
		if len(originalApp.Ext) > 0 {
			newApp.Ext, err = jsonpatch.MergePatch(originalApp.Ext, fpdConfigApp.Ext)
		} else {
			newApp.Ext = fpdConfigApp.Ext
		}
	}

	return newApp, err
}

func buildExtData(data []byte) []byte {
	res := make([]byte, 0, len(data))
	res = append(res, []byte(`{"data":`)...)
	res = append(res, data...)
	res = append(res, []byte(`}`)...)
	return res
}

func ExtractBidderConfigFPD(reqExtPrebid openrtb_ext.ExtRequestPrebid) (map[openrtb_ext.BidderName]*openrtb_ext.ORTB2, openrtb_ext.ExtRequestPrebid) {
	//map to store bidder configs to process
	fpdData := make(map[openrtb_ext.BidderName]*openrtb_ext.ORTB2)

	//every bidder in ext.prebid.data.bidders should receive fpd data if defined
	bidderTable := make(map[string]struct{}) //just need to check existence of the element in map

	if reqExtPrebid.BidderConfigs != nil {
		for _, bidderConfig := range *reqExtPrebid.BidderConfigs {
			for _, bidder := range bidderConfig.Bidders {

				if _, present := bidderTable[bidder]; !present {
					bidderTable[bidder] = struct{}{}
					fpdData[openrtb_ext.BidderName(bidder)] = &openrtb_ext.ORTB2{}
				}
				//this will overwrite previously set site/app/user.
				//Last defined bidder-specific config will take precedence
				fpdBidderData := fpdData[openrtb_ext.BidderName(bidder)]
				if bidderConfig.Config != nil && bidderConfig.Config.ORTB2 != nil {
					if bidderConfig.Config.ORTB2.Site != nil {
						fpdBidderData.Site = bidderConfig.Config.ORTB2.Site
					}
					if bidderConfig.Config.ORTB2.App != nil {
						fpdBidderData.App = bidderConfig.Config.ORTB2.App
					}
					if bidderConfig.Config.ORTB2.User != nil {
						fpdBidderData.User = bidderConfig.Config.ORTB2.User
					}
				}

			}
		}
	}

	reqExtPrebid.BidderConfigs = nil

	return fpdData, reqExtPrebid
}
