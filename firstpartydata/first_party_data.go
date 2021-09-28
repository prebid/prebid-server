package firstpartydata

import (
	"encoding/json"
	"fmt"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
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

type ResolvedFirstPartyData struct {
	Site *openrtb2.Site
	App  *openrtb2.App
	User *openrtb2.User
}

//ExtractGlobalFPD collect it and remove req.{site,app,user}.ext.data if exists
func ExtractGlobalFPD(req *openrtb_ext.RequestWrapper) (map[string][]byte, error) {

	fpdReqData := make(map[string][]byte, 3)

	siteExt, err := req.GetSiteExt()
	if err != nil {
		return nil, err
	}

	if len(siteExt.GetExt()[dataKey]) > 0 {
		newSiteExt := siteExt.GetExt()
		fpdReqData[siteKey] = newSiteExt[dataKey]
		delete(newSiteExt, dataKey)
		siteExt.SetExt(newSiteExt)
	}

	appExt, err := req.GetAppExt()
	if err != nil {
		return nil, err
	}
	if len(appExt.GetExt()[dataKey]) > 0 {
		newAppExt := appExt.GetExt()
		fpdReqData[appKey] = newAppExt[dataKey]
		delete(newAppExt, dataKey)
		appExt.SetExt(newAppExt)
	}

	userExt, err := req.GetUserExt()
	if err != nil {
		return nil, err
	}
	if len(userExt.GetExt()[dataKey]) > 0 {
		newUserExt := userExt.GetExt()
		fpdReqData[userKey] = newUserExt[dataKey]
		delete(newUserExt, dataKey)
		userExt.SetExt(newUserExt)
	}

	return fpdReqData, nil
}

//ExtractOpenRtbGlobalFPD extracts and deletes user.data and {app/site}.content.data from request
func ExtractOpenRtbGlobalFPD(bidRequest *openrtb2.BidRequest) map[string][]openrtb2.Data {

	openRtbGlobalFPD := make(map[string][]openrtb2.Data, 3)
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

//ResolveFPD consolidates First Party Data from different sources and returns valid FPD that will be applied to bidders later or returns errors
func ResolveFPD(bidRequest *openrtb2.BidRequest, fpdBidderConfigData map[openrtb_ext.BidderName]*openrtb_ext.ORTB2, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, biddersWithGlobalFPD []string) (map[openrtb_ext.BidderName]*ResolvedFirstPartyData, []error) {
	var errL []error

	resolvedFpd := make(map[openrtb_ext.BidderName]*ResolvedFirstPartyData)

	//convert list to map to optimize check if value exists
	globalBiddersTable := make(map[string]struct{}) //just need to check existence of the element in map
	for _, bidderName := range biddersWithGlobalFPD {
		globalBiddersTable[bidderName] = struct{}{}
	}

	for bName, fpdConfig := range fpdBidderConfigData {
		bidderName := string(bName)

		_, hasGlobalFPD := globalBiddersTable[bidderName]

		resolvedFpdConfig := &ResolvedFirstPartyData{}

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
		resolvedFpdConfig.Site = newSite

		if len(errL) == 0 {
			resolvedFpd[bName] = resolvedFpdConfig
		}
	}
	return resolvedFpd, errL
}

func resolveUser(fpdConfigUser *map[string]json.RawMessage, bidRequestUser *openrtb2.User, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool, bidderName string) (*openrtb2.User, error) {

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
		newUser, err = mergeUsers(&newUser, *fpdConfigUser)
	}

	return &newUser, err
}

func unmarshalJSONToInt64(b json.RawMessage) (int64, error) {
	var num json.Number
	err := json.Unmarshal(b, &num)
	if err != nil {
		return -1, err
	}
	resNum, err := num.Int64()
	return resNum, err
}

//resolveExtension inserts remaining {site/app/user} attributes back to {site/app/user}.ext.data
func resolveExtension(fpdConfig map[string]json.RawMessage, originalExt json.RawMessage) ([]byte, error) {
	resExt := originalExt
	var err error

	fpdConfigExt, present := fpdConfig["ext"]
	if present {
		delete(fpdConfig, "ext")
		resExt, err = jsonpatch.MergePatch(resExt, fpdConfigExt)
		if err != nil {
			return nil, err
		}
	}

	if len(fpdConfig) > 0 {
		fpdData, err := json.Marshal(fpdConfig)
		if err != nil {
			return nil, err
		}
		data := buildExtData(fpdData)
		return jsonpatch.MergePatch(resExt, data)
	}
	return resExt, nil
}

func mergeUsers(original *openrtb2.User, fpdConfigUser map[string]json.RawMessage) (openrtb2.User, error) {

	var err error
	newUser := *original

	if keywords, present := fpdConfigUser["keywords"]; present {
		newUser.Keywords = string(keywords)
		delete(fpdConfigUser, "keywords")
	}
	if gender, present := fpdConfigUser["gender"]; present {
		newUser.Gender = string(gender)
		delete(fpdConfigUser, "gender")
	}
	if yob, present := fpdConfigUser["yob"]; present {
		yobNum, err := unmarshalJSONToInt64(yob)
		if err != nil {
			return newUser, err
		}
		newUser.Yob = yobNum
		delete(fpdConfigUser, "yob")
	}

	if len(fpdConfigUser) > 0 {
		newUser.Ext, err = resolveExtension(fpdConfigUser, original.Ext)
	}

	return newUser, err
}

func resolveSite(fpdConfigSite *map[string]json.RawMessage, bidRequestSite *openrtb2.Site, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool, bidderName string) (*openrtb2.Site, error) {

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
				newSite.Content = &openrtb2.Content{Data: openRtbGlobalFPD[siteContentDataKey]}
			}
		}
	}

	if fpdConfigSite != nil {
		newSite, err = mergeSites(&newSite, *fpdConfigSite, bidderName)
	}
	return &newSite, err

}

func mergeSites(originalSite *openrtb2.Site, fpdConfigSite map[string]json.RawMessage, bidderName string) (openrtb2.Site, error) {

	var err error
	newSite := *originalSite

	if page, present := fpdConfigSite["page"]; present {
		sitePage := string(page)
		//apply bidder specific fpd if present
		//result site should have ID or Page, fpd becomes incorrect if it overwrites page to empty one and ID is empty in original site
		if sitePage == "" && newSite.Page != "" && newSite.ID == "" {
			return newSite, fmt.Errorf("incorrect First Party Data for bidder %s: Site object cannot set empty page if req.site.id is empty", bidderName)

		}
		newSite.Page = sitePage
		delete(fpdConfigSite, "page")
	}
	if name, present := fpdConfigSite["name"]; present {
		newSite.Name = string(name)
		delete(fpdConfigSite, "name")
	}
	if domain, present := fpdConfigSite["domain"]; present {
		newSite.Domain = string(domain)
		delete(fpdConfigSite, "domain")
	}
	if cat, present := fpdConfigSite["cat"]; present {
		var siteCat []string
		err := json.Unmarshal(cat, &siteCat)
		if err != nil {
			return newSite, err
		}
		newSite.Cat = siteCat
		delete(fpdConfigSite, "cat")
	}
	if sectionCat, present := fpdConfigSite["sectioncat"]; present {
		var siteSectionCat []string
		err := json.Unmarshal(sectionCat, &siteSectionCat)
		if err != nil {
			return newSite, err
		}
		newSite.SectionCat = siteSectionCat
		delete(fpdConfigSite, "sectioncat")
	}
	if pageCat, present := fpdConfigSite["pagecat"]; present {
		var sitePageCat []string
		err := json.Unmarshal(pageCat, &sitePageCat)
		if err != nil {
			return newSite, err
		}
		newSite.PageCat = sitePageCat
		delete(fpdConfigSite, "pagecat")
	}
	if search, present := fpdConfigSite["search"]; present {
		newSite.Search = string(search)
		delete(fpdConfigSite, "search")
	}
	if keywords, present := fpdConfigSite["keywords"]; present {
		newSite.Keywords = string(keywords)
		delete(fpdConfigSite, "keywords")
	}
	if ref, present := fpdConfigSite["ref"]; present {
		newSite.Ref = string(ref)
		delete(fpdConfigSite, "ref")
	}

	if len(fpdConfigSite) > 0 {
		newSite.Ext, err = resolveExtension(fpdConfigSite, originalSite.Ext)
	}

	return newSite, err
}

func resolveApp(fpdConfigApp *map[string]json.RawMessage, bidRequestApp *openrtb2.App, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, hasGlobalFPD bool, bidderName string) (*openrtb2.App, error) {

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
				newApp.Content = &openrtb2.Content{Data: openRtbGlobalFPD[appContentDataKey]}
			}
		}
	}

	if fpdConfigApp != nil {
		//apply bidder specific fpd if present
		newApp, err = mergeApps(&newApp, *fpdConfigApp)
	}

	return &newApp, err
}

func mergeApps(originalApp *openrtb2.App, fpdConfigApp map[string]json.RawMessage) (openrtb2.App, error) {

	var err error
	newApp := *originalApp

	if name, present := fpdConfigApp["name"]; present {
		newApp.Name = string(name)
		delete(fpdConfigApp, "name")
	}
	if bundle, present := fpdConfigApp["bundle"]; present {
		newApp.Bundle = string(bundle)
		delete(fpdConfigApp, "bundle")
	}
	if domain, present := fpdConfigApp["domain"]; present {
		newApp.Domain = string(domain)
		delete(fpdConfigApp, "domain")
	}
	if storeurl, present := fpdConfigApp["storeurl"]; present {
		newApp.StoreURL = string(storeurl)
		delete(fpdConfigApp, "storeurl")
	}
	if cat, present := fpdConfigApp["cat"]; present {
		var siteCat []string
		err := json.Unmarshal(cat, &siteCat)
		if err != nil {
			return newApp, err
		}
		newApp.Cat = siteCat
		delete(fpdConfigApp, "cat")
	}
	if sectionCat, present := fpdConfigApp["sectioncat"]; present {
		var siteSectionCat []string
		err := json.Unmarshal(sectionCat, &siteSectionCat)
		if err != nil {
			return newApp, err
		}
		newApp.SectionCat = siteSectionCat
		delete(fpdConfigApp, "sectioncat")
	}
	if pageCat, present := fpdConfigApp["pagecat"]; present {
		var sitePageCat []string
		err := json.Unmarshal(pageCat, &sitePageCat)
		if err != nil {
			return newApp, err
		}
		newApp.PageCat = sitePageCat
		delete(fpdConfigApp, "pagecat")
	}
	if keywords, present := fpdConfigApp["ver"]; present {
		newApp.Ver = string(keywords)
		delete(fpdConfigApp, "ver")
	}
	if keywords, present := fpdConfigApp["keywords"]; present {
		newApp.Keywords = string(keywords)
		delete(fpdConfigApp, "keywords")
	}

	if len(fpdConfigApp) > 0 {
		newApp.Ext, err = resolveExtension(fpdConfigApp, originalApp.Ext)
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

//ExtractBidderConfigFPD extracts bidder specific configs from req.ext.prebid.bidderconfig
func ExtractBidderConfigFPD(reqExt *openrtb_ext.RequestExt) map[openrtb_ext.BidderName]*openrtb_ext.ORTB2 {

	fpd := make(map[openrtb_ext.BidderName]*openrtb_ext.ORTB2)
	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid != nil && reqExtPrebid.BidderConfigs != nil {
		for _, bidderConfig := range *reqExtPrebid.BidderConfigs {
			for _, bidder := range bidderConfig.Bidders {

				if _, present := fpd[openrtb_ext.BidderName(bidder)]; !present {
					fpd[openrtb_ext.BidderName(bidder)] = &openrtb_ext.ORTB2{}
				}
				//this will overwrite previously set site/app/user.
				//Last defined bidder-specific config will take precedence
				fpdBidderData := fpd[openrtb_ext.BidderName(bidder)]

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
		reqExtPrebid.BidderConfigs = nil
		reqExt.SetPrebid(reqExtPrebid)
	}
	return fpd

}

//ExtractFPDForBidders extracts FPD data from request if specified
func ExtractFPDForBidders(req *openrtb_ext.RequestWrapper) (map[openrtb_ext.BidderName]*ResolvedFirstPartyData, []error) {

	reqExt, err := req.GetRequestExt()
	if err != nil {
		return nil, []error{err}
	}
	if reqExt == nil || reqExt.GetPrebid() == nil {
		return nil, nil
	}
	var biddersWithGlobalFPD []string

	extPrebid := reqExt.GetPrebid()
	if prebidData := extPrebid.Data; prebidData != nil {
		biddersWithGlobalFPD = prebidData.Bidders
		extPrebid.Data.Bidders = nil
		reqExt.SetPrebid(extPrebid)
	}

	fbdBidderConfigData := ExtractBidderConfigFPD(reqExt)

	var globalFpd map[string][]byte
	// if global bidder list is nill (different from empty list!)
	// or doesn't exists - don't remove {site/app/user}.ext.data from request
	if biddersWithGlobalFPD != nil {
		globalFpd, err = ExtractGlobalFPD(req)
		if err != nil {
			return nil, []error{err}
		}
	}

	if len(fbdBidderConfigData) == 0 && len(biddersWithGlobalFPD) == 0 {
		return nil, nil
	}

	//If ext.prebid.data.bidders isn't defined, the default is there's no permission filtering
	openRtbGlobalFPD := ExtractOpenRtbGlobalFPD(req.BidRequest)

	return ResolveFPD(req.BidRequest, fbdBidderConfigData, globalFpd, openRtbGlobalFPD, biddersWithGlobalFPD)

}
