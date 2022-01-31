package firstpartydata

import (
	"encoding/json"
	"fmt"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
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

	keywordsKey   = "keywords"
	genderKey     = "gender"
	yobKey        = "yob"
	pageKey       = "page"
	nameKey       = "name"
	domainKey     = "domain"
	catKey        = "cat"
	sectionCatKey = "sectioncat"
	pageCatKey    = "pagecat"
	searchKey     = "search"
	refKey        = "ref"
	bundleKey     = "bundle"
	storeUrlKey   = "storeurl"
	verKey        = "ver"
)

type ResolvedFirstPartyData struct {
	Site *openrtb2.Site
	App  *openrtb2.App
	User *openrtb2.User
}

// ExtractGlobalFPD extracts request level FPD from the request and removes req.{site,app,user}.ext.data if exists
func ExtractGlobalFPD(req *openrtb_ext.RequestWrapper) (map[string][]byte, error) {

	fpdReqData := make(map[string][]byte, 3)

	siteExt, err := req.GetSiteExt()
	if err != nil {
		return nil, err
	}
	refreshExt := false

	if len(siteExt.GetExt()[dataKey]) > 0 {
		newSiteExt := siteExt.GetExt()
		fpdReqData[siteKey] = newSiteExt[dataKey]
		delete(newSiteExt, dataKey)
		siteExt.SetExt(newSiteExt)
		refreshExt = true
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
		refreshExt = true
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
		refreshExt = true
	}
	if refreshExt {
		//need to keep site/app/user ext clean in case bidder is not in global fpd bidder list
		// rebuild/resync the request in the request wrapper.
		if err := req.RebuildRequest(); err != nil {
			return nil, err
		}
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

	allBiddersTable := make(map[string]struct{})

	if biddersWithGlobalFPD == nil {
		//add all bidders in bidder configs to receive global data and bidder specific data
		for bidderName := range fpdBidderConfigData {
			if _, present := allBiddersTable[string(bidderName)]; !present {
				allBiddersTable[string(bidderName)] = struct{}{}
			}
		}
	} else {
		//only bidders in global bidder list will receive global data and bidder specific data
		for _, bidderName := range biddersWithGlobalFPD {
			if _, present := allBiddersTable[string(bidderName)]; !present {
				allBiddersTable[string(bidderName)] = struct{}{}
			}
		}
	}

	for bidderName := range allBiddersTable {

		fpdConfig := fpdBidderConfigData[openrtb_ext.BidderName(bidderName)]

		resolvedFpdConfig := &ResolvedFirstPartyData{}

		newUser, err := resolveUser(fpdConfig, bidRequest.User, globalFPD, openRtbGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.User = newUser

		newApp, err := resolveApp(fpdConfig, bidRequest.App, globalFPD, openRtbGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.App = newApp

		newSite, err := resolveSite(fpdConfig, bidRequest.Site, globalFPD, openRtbGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.Site = newSite

		if len(errL) == 0 {
			resolvedFpd[openrtb_ext.BidderName(bidderName)] = resolvedFpdConfig
		}
	}
	return resolvedFpd, errL
}

func resolveUser(fpdConfig *openrtb_ext.ORTB2, bidRequestUser *openrtb2.User, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, bidderName string) (*openrtb2.User, error) {
	var fpdConfigUser map[string]json.RawMessage

	if fpdConfig != nil && fpdConfig.User != nil {
		fpdConfigUser = fpdConfig.User
	}

	if bidRequestUser == nil && fpdConfigUser == nil {
		return nil, nil
	}

	if bidRequestUser == nil && fpdConfigUser != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("incorrect First Party Data for bidder %s: User object is not defined in request, but defined in FPD config", bidderName),
		}
	}

	newUser := *bidRequestUser
	var err error

	//apply global fpd
	if len(globalFPD[userKey]) > 0 {
		extData := buildExtData(globalFPD[userKey])
		if len(newUser.Ext) > 0 {
			newUser.Ext, err = jsonpatch.MergePatch(newUser.Ext, extData)
		} else {
			newUser.Ext = extData
		}
	}
	if openRtbGlobalFPD != nil && len(openRtbGlobalFPD[userDataKey]) > 0 {
		newUser.Data = openRtbGlobalFPD[userDataKey]
	}
	if fpdConfigUser != nil {
		//apply bidder specific fpd if present
		newUser, err = mergeUsers(&newUser, fpdConfigUser)
	}

	return &newUser, err
}

func unmarshalJSONToInt64(input json.RawMessage) (int64, error) {
	var num json.Number
	err := json.Unmarshal(input, &num)
	if err != nil {
		return -1, err
	}
	resNum, err := num.Int64()
	return resNum, err
}

func unmarshalJSONToString(input json.RawMessage) (string, error) {
	var inputString string
	err := json.Unmarshal(input, &inputString)
	return inputString, err
}

func unmarshalJSONToStringArray(input json.RawMessage) ([]string, error) {
	var inputString []string
	err := json.Unmarshal(input, &inputString)
	return inputString, err
}

//resolveExtension inserts remaining {site/app/user} attributes back to {site/app/user}.ext.data
func resolveExtension(fpdConfig map[string]json.RawMessage, originalExt json.RawMessage) ([]byte, error) {
	resExt := originalExt
	var err error

	if resExt == nil && len(fpdConfig) > 0 {
		fpdExt, err := json.Marshal(fpdConfig)
		return buildExtData(fpdExt), err
	}

	fpdConfigExt, present := fpdConfig[extKey]
	if present {
		delete(fpdConfig, extKey)
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

	if keywords, present := fpdConfigUser[keywordsKey]; present {
		newUser.Keywords, err = unmarshalJSONToString(keywords)
		if err != nil {
			return newUser, err
		}
		delete(fpdConfigUser, keywordsKey)
	}
	if gender, present := fpdConfigUser[genderKey]; present {
		newUser.Gender, err = unmarshalJSONToString(gender)
		if err != nil {
			return newUser, err
		}
		delete(fpdConfigUser, genderKey)
	}
	if yob, present := fpdConfigUser[yobKey]; present {
		newUser.Yob, err = unmarshalJSONToInt64(yob)
		if err != nil {
			return newUser, err
		}
		delete(fpdConfigUser, yobKey)
	}

	if len(fpdConfigUser) > 0 {
		newUser.Ext, err = resolveExtension(fpdConfigUser, original.Ext)
	}

	return newUser, err
}

func resolveSite(fpdConfig *openrtb_ext.ORTB2, bidRequestSite *openrtb2.Site, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, bidderName string) (*openrtb2.Site, error) {
	var fpdConfigSite map[string]json.RawMessage

	if fpdConfig != nil && fpdConfig.Site != nil {
		fpdConfigSite = fpdConfig.Site
	}

	if bidRequestSite == nil && fpdConfigSite == nil {
		return nil, nil
	}
	if bidRequestSite == nil && fpdConfigSite != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("incorrect First Party Data for bidder %s: Site object is not defined in request, but defined in FPD config", bidderName),
		}
	}

	newSite := *bidRequestSite
	var err error

	//apply global fpd
	if len(globalFPD[siteKey]) > 0 {
		extData := buildExtData(globalFPD[siteKey])
		if len(newSite.Ext) > 0 {
			newSite.Ext, err = jsonpatch.MergePatch(newSite.Ext, extData)
		} else {
			newSite.Ext = extData
		}
	}
	if openRtbGlobalFPD != nil && len(openRtbGlobalFPD[siteContentDataKey]) > 0 {
		if newSite.Content != nil {
			contentCopy := *newSite.Content
			contentCopy.Data = openRtbGlobalFPD[siteContentDataKey]
			newSite.Content = &contentCopy
		} else {
			newSite.Content = &openrtb2.Content{Data: openRtbGlobalFPD[siteContentDataKey]}
		}
	}

	if fpdConfigSite != nil {
		newSite, err = mergeSites(&newSite, fpdConfigSite, bidderName)
	}
	return &newSite, err

}
func mergeSites(originalSite *openrtb2.Site, fpdConfigSite map[string]json.RawMessage, bidderName string) (openrtb2.Site, error) {

	var err error
	newSite := *originalSite

	if page, present := fpdConfigSite[pageKey]; present {
		sitePage, err := unmarshalJSONToString(page)
		if err != nil {
			return newSite, err
		}
		//apply bidder specific fpd if present
		//result site should have ID or Page, fpd becomes incorrect if it overwrites page to empty one and ID is empty in original site
		if sitePage == "" && newSite.Page != "" && newSite.ID == "" {
			return newSite, &errortypes.BadInput{
				Message: fmt.Sprintf("incorrect First Party Data for bidder %s: Site object cannot set empty page if req.site.id is empty", bidderName),
			}

		}
		newSite.Page = sitePage
		delete(fpdConfigSite, pageKey)
	}
	if name, present := fpdConfigSite[nameKey]; present {
		newSite.Name, err = unmarshalJSONToString(name)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, nameKey)
	}
	if domain, present := fpdConfigSite[domainKey]; present {
		newSite.Domain, err = unmarshalJSONToString(domain)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, domainKey)
	}
	if cat, present := fpdConfigSite[catKey]; present {
		newSite.Cat, err = unmarshalJSONToStringArray(cat)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, catKey)
	}
	if sectionCat, present := fpdConfigSite[sectionCatKey]; present {
		newSite.SectionCat, err = unmarshalJSONToStringArray(sectionCat)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, sectionCatKey)
	}
	if pageCat, present := fpdConfigSite[pageCatKey]; present {
		newSite.PageCat, err = unmarshalJSONToStringArray(pageCat)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, pageCatKey)
	}
	if search, present := fpdConfigSite[searchKey]; present {
		newSite.Search, err = unmarshalJSONToString(search)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, searchKey)
	}
	if keywords, present := fpdConfigSite[keywordsKey]; present {
		newSite.Keywords, err = unmarshalJSONToString(keywords)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, keywordsKey)
	}
	if ref, present := fpdConfigSite[refKey]; present {
		newSite.Ref, err = unmarshalJSONToString(ref)
		if err != nil {
			return newSite, err
		}
		delete(fpdConfigSite, refKey)
	}

	if len(fpdConfigSite) > 0 {
		newSite.Ext, err = resolveExtension(fpdConfigSite, originalSite.Ext)
	}

	return newSite, err
}

func resolveApp(fpdConfig *openrtb_ext.ORTB2, bidRequestApp *openrtb2.App, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, bidderName string) (*openrtb2.App, error) {

	var fpdConfigApp map[string]json.RawMessage

	if fpdConfig != nil && fpdConfig.App != nil {
		fpdConfigApp = fpdConfig.App
	}

	if bidRequestApp == nil && fpdConfigApp == nil {
		return nil, nil
	}

	if bidRequestApp == nil && fpdConfigApp != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("incorrect First Party Data for bidder %s: App object is not defined in request, but defined in FPD config", bidderName),
		}
	}

	newApp := *bidRequestApp
	var err error

	//apply global fpd if exists
	if len(globalFPD[appKey]) > 0 {
		extData := buildExtData(globalFPD[appKey])
		if len(newApp.Ext) > 0 {
			newApp.Ext, err = jsonpatch.MergePatch(newApp.Ext, extData)
		} else {
			newApp.Ext = extData
		}
	}
	if openRtbGlobalFPD != nil && len(openRtbGlobalFPD[appContentDataKey]) > 0 {
		if newApp.Content != nil {
			contentCopy := *newApp.Content
			contentCopy.Data = openRtbGlobalFPD[appContentDataKey]
			newApp.Content = &contentCopy
		} else {
			newApp.Content = &openrtb2.Content{Data: openRtbGlobalFPD[appContentDataKey]}
		}
	}

	if fpdConfigApp != nil {
		//apply bidder specific fpd if present
		newApp, err = mergeApps(&newApp, fpdConfigApp)
	}

	return &newApp, err
}

func mergeApps(originalApp *openrtb2.App, fpdConfigApp map[string]json.RawMessage) (openrtb2.App, error) {

	var err error
	newApp := *originalApp

	if name, present := fpdConfigApp[nameKey]; present {
		newApp.Name, err = unmarshalJSONToString(name)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, nameKey)
	}
	if bundle, present := fpdConfigApp[bundleKey]; present {
		newApp.Bundle, err = unmarshalJSONToString(bundle)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, bundleKey)
	}
	if domain, present := fpdConfigApp[domainKey]; present {
		newApp.Domain, err = unmarshalJSONToString(domain)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, domainKey)
	}
	if storeUrl, present := fpdConfigApp[storeUrlKey]; present {
		newApp.StoreURL, err = unmarshalJSONToString(storeUrl)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, storeUrlKey)
	}
	if cat, present := fpdConfigApp[catKey]; present {
		newApp.Cat, err = unmarshalJSONToStringArray(cat)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, catKey)
	}
	if sectionCat, present := fpdConfigApp[sectionCatKey]; present {
		newApp.SectionCat, err = unmarshalJSONToStringArray(sectionCat)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, sectionCatKey)
	}
	if pageCat, present := fpdConfigApp[pageCatKey]; present {
		newApp.PageCat, err = unmarshalJSONToStringArray(pageCat)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, pageCatKey)
	}
	if version, present := fpdConfigApp[verKey]; present {
		newApp.Ver, err = unmarshalJSONToString(version)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, verKey)
	}
	if keywords, present := fpdConfigApp[keywordsKey]; present {
		newApp.Keywords, err = unmarshalJSONToString(keywords)
		if err != nil {
			return newApp, err
		}
		delete(fpdConfigApp, keywordsKey)
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
func ExtractBidderConfigFPD(reqExt *openrtb_ext.RequestExt) (map[openrtb_ext.BidderName]*openrtb_ext.ORTB2, error) {

	fpd := make(map[openrtb_ext.BidderName]*openrtb_ext.ORTB2)
	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid != nil {
		for _, bidderConfig := range reqExtPrebid.BidderConfigs {
			for _, bidder := range bidderConfig.Bidders {
				if _, present := fpd[openrtb_ext.BidderName(bidder)]; present {
					//if bidder has duplicated config - throw an error
					return nil, &errortypes.BadInput{
						Message: fmt.Sprintf("multiple First Party Data bidder configs provided for bidder: %s", bidder),
					}
				}

				fpdBidderData := &openrtb_ext.ORTB2{}

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

				fpd[openrtb_ext.BidderName(bidder)] = fpdBidderData
			}
		}
		reqExtPrebid.BidderConfigs = nil
		reqExt.SetPrebid(reqExtPrebid)
	}
	return fpd, nil

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
	if extPrebid.Data != nil {
		biddersWithGlobalFPD = extPrebid.Data.Bidders
		extPrebid.Data.Bidders = nil
		reqExt.SetPrebid(extPrebid)
	}

	fbdBidderConfigData, err := ExtractBidderConfigFPD(reqExt)
	if err != nil {
		return nil, []error{err}
	}

	var globalFpd map[string][]byte
	var openRtbGlobalFPD map[string][]openrtb2.Data

	if biddersWithGlobalFPD != nil {
		//global fpd data should not be extracted and removed from request if global bidder list is nil.
		//Bidders that don't have any fpd config should receive request data as is
		globalFpd, err = ExtractGlobalFPD(req)
		if err != nil {
			return nil, []error{err}
		}
		openRtbGlobalFPD = ExtractOpenRtbGlobalFPD(req.BidRequest)
	}

	return ResolveFPD(req.BidRequest, fbdBidderConfigData, globalFpd, openRtbGlobalFPD, biddersWithGlobalFPD)

}
