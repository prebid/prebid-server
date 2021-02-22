package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/prebid/go-gdpr/vendorconsent"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/lmt"
)

var integrationTypeMap = map[metrics.RequestType]config.IntegrationType{
	metrics.ReqTypeAMP:      config.IntegrationTypeAMP,
	metrics.ReqTypeORTB2App: config.IntegrationTypeApp,
	metrics.ReqTypeVideo:    config.IntegrationTypeVideo,
	metrics.ReqTypeORTB2Web: config.IntegrationTypeWeb,
}

const unknownBidder string = ""

func BidderToPrebidSChains(req *openrtb_ext.ExtRequest) (map[string]*openrtb_ext.ExtRequestPrebidSChainSChain, error) {
	bidderToSChains := make(map[string]*openrtb_ext.ExtRequestPrebidSChainSChain)

	if req != nil {
		for _, schainWrapper := range req.Prebid.SChains {
			for _, bidder := range schainWrapper.Bidders {
				if _, present := bidderToSChains[bidder]; present {
					return nil, fmt.Errorf("request.ext.prebid.schains contains multiple schains for bidder %s; "+
						"it must contain no more than one per bidder.", bidder)
				} else {
					bidderToSChains[bidder] = &schainWrapper.SChain
				}
			}
		}
	}

	return bidderToSChains, nil
}

// cleanOpenRTBRequests splits the input request into requests which are sanitized for each bidder. Intended behavior is:
//
//   1. BidRequest.Imp[].Ext will only contain the "prebid" field and a "bidder" field which has the params for the intended Bidder.
//   2. Every BidRequest.Imp[] requested Bids from the Bidder who keys it.
//   3. BidRequest.User.BuyerUID will be set to that Bidder's ID.
func cleanOpenRTBRequests(ctx context.Context,
	req AuctionRequest,
	requestExt *openrtb_ext.ExtRequest,
	gDPR gdpr.Permissions,
	usersyncIfAmbiguous bool,
	privacyConfig config.Privacy) (bidderRequests []BidderRequest, privacyLabels metrics.PrivacyLabels, errs []error) {

	impsByBidder, errs := splitImps(req.BidRequest.Imp)
	if len(errs) > 0 {
		return
	}

	aliases, errs := parseAliases(req.BidRequest)
	if len(errs) > 0 {
		return
	}

	bidderRequests, errs = getAuctionBidderRequests(req, requestExt, impsByBidder, aliases)

	if len(bidderRequests) == 0 {
		return
	}

	gdprSignal, err := extractGDPR(req.BidRequest)
	if err != nil {
		errs = append(errs, err)
	}
	consent, err := extractConsent(req.BidRequest)
	if err != nil {
		errs = append(errs, err)
	}
	gdprEnforced := gdprSignal == gdpr.SignalYes || (gdprSignal == gdpr.SignalAmbiguous && !usersyncIfAmbiguous)

	ccpaEnforcer, err := extractCCPA(req.BidRequest, privacyConfig, &req.Account, aliases, integrationTypeMap[req.LegacyLabels.RType])
	if err != nil {
		errs = append(errs, err)
	}

	lmtEnforcer := extractLMT(req.BidRequest, privacyConfig)

	// request level privacy policies
	privacyEnforcement := privacy.Enforcement{
		COPPA: req.BidRequest.Regs != nil && req.BidRequest.Regs.COPPA == 1,
		LMT:   lmtEnforcer.ShouldEnforce(unknownBidder),
	}

	privacyLabels.CCPAProvided = ccpaEnforcer.CanEnforce()
	privacyLabels.CCPAEnforced = ccpaEnforcer.ShouldEnforce(unknownBidder)
	privacyLabels.COPPAEnforced = privacyEnforcement.COPPA
	privacyLabels.LMTEnforced = lmtEnforcer.ShouldEnforce(unknownBidder)

	gdprEnforced = gdprEnforced && gdprEnabled(&req.Account, privacyConfig, integrationTypeMap[req.LegacyLabels.RType])

	if gdprEnforced {
		privacyLabels.GDPREnforced = true
		parsedConsent, err := vendorconsent.ParseString(consent)
		if err == nil {
			version := int(parsedConsent.Version())
			privacyLabels.GDPRTCFVersion = metrics.TCFVersionToValue(version)
		}
	}

	// bidder level privacy policies
	for _, bidderRequest := range bidderRequests {
		// CCPA
		privacyEnforcement.CCPA = ccpaEnforcer.ShouldEnforce(bidderRequest.BidderName.String())

		// GDPR
		if gdprEnforced {
			var publisherID = req.LegacyLabels.PubID
			_, geo, id, err := gDPR.PersonalInfoAllowed(ctx, bidderRequest.BidderCoreName, publisherID, gdprSignal, consent)
			if err == nil {
				privacyEnforcement.GDPRGeo = !geo
				privacyEnforcement.GDPRID = !id
			} else {
				privacyEnforcement.GDPRGeo = true
				privacyEnforcement.GDPRID = true
			}
		}

		privacyEnforcement.Apply(bidderRequest.BidRequest)
	}

	return
}

func gdprEnabled(account *config.Account, privacyConfig config.Privacy, integrationType config.IntegrationType) bool {
	if accountEnabled := account.GDPR.EnabledForIntegrationType(integrationType); accountEnabled != nil {
		return *accountEnabled
	}
	return privacyConfig.GDPR.Enabled
}

func ccpaEnabled(account *config.Account, privacyConfig config.Privacy, requestType config.IntegrationType) bool {
	if accountEnabled := account.CCPA.EnabledForIntegrationType(requestType); accountEnabled != nil {
		return *accountEnabled
	}
	return privacyConfig.CCPA.Enforce
}

func extractCCPA(orig *openrtb.BidRequest, privacyConfig config.Privacy, account *config.Account, aliases map[string]string, requestType config.IntegrationType) (privacy.PolicyEnforcer, error) {
	ccpaPolicy, err := ccpa.ReadFromRequest(orig)
	if err != nil {
		return privacy.NilPolicyEnforcer{}, err
	}

	validBidders := GetValidBidders(aliases)
	ccpaParsedPolicy, err := ccpaPolicy.Parse(validBidders)
	if err != nil {
		return privacy.NilPolicyEnforcer{}, err
	}

	ccpaEnforcer := privacy.EnabledPolicyEnforcer{
		Enabled:        ccpaEnabled(account, privacyConfig, requestType),
		PolicyEnforcer: ccpaParsedPolicy,
	}
	return ccpaEnforcer, nil
}

func extractLMT(orig *openrtb.BidRequest, privacyConfig config.Privacy) privacy.PolicyEnforcer {
	return privacy.EnabledPolicyEnforcer{
		Enabled:        privacyConfig.LMT.Enforce,
		PolicyEnforcer: lmt.ReadFromRequest(orig),
	}
}

func getAuctionBidderRequests(req AuctionRequest,
	requestExt *openrtb_ext.ExtRequest,
	impsByBidder map[string][]openrtb.Imp,
	aliases map[string]string) ([]BidderRequest, []error) {

	bidderRequests := make([]BidderRequest, 0, len(impsByBidder))

	explicitBuyerUIDs, err := extractBuyerUIDs(req.BidRequest.User)
	if err != nil {
		return nil, []error{err}
	}

	var sChainsByBidder map[string]*openrtb_ext.ExtRequestPrebidSChainSChain

	sChainsByBidder, err = BidderToPrebidSChains(requestExt)
	if err != nil {
		return nil, []error{err}
	}

	reqExt, err := getExtJson(req.BidRequest, requestExt)
	if err != nil {
		return nil, []error{err}
	}

	var errs []error
	for bidder, imps := range impsByBidder {
		coreBidder := resolveBidder(bidder, aliases)

		reqCopy := *req.BidRequest
		reqCopy.Imp = imps
		reqCopy.Ext = reqExt

		prepareSource(&reqCopy, bidder, sChainsByBidder)

		if err := removeUnpermissionedEids(&reqCopy, bidder, requestExt); err != nil {
			errs = append(errs, fmt.Errorf("unable to enforce request.ext.prebid.data.eidpermissions because %v", err))
			continue
		}

		bidderRequest := BidderRequest{
			BidderName:     openrtb_ext.BidderName(bidder),
			BidderCoreName: coreBidder,
			BidRequest:     &reqCopy,
			BidderLabels: metrics.AdapterLabels{
				Source:      req.LegacyLabels.Source,
				RType:       req.LegacyLabels.RType,
				Adapter:     coreBidder,
				PubID:       req.LegacyLabels.PubID,
				CookieFlag:  req.LegacyLabels.CookieFlag,
				AdapterBids: metrics.AdapterBidPresent,
			},
		}

		if hadSync := prepareUser(&reqCopy, bidder, coreBidder, explicitBuyerUIDs, req.UserSyncs); !hadSync && req.BidRequest.App == nil {
			bidderRequest.BidderLabels.CookieFlag = metrics.CookieFlagNo
		} else {
			bidderRequest.BidderLabels.CookieFlag = metrics.CookieFlagYes
		}

		bidderRequests = append(bidderRequests, bidderRequest)
	}
	return bidderRequests, errs
}

func getExtJson(req *openrtb.BidRequest, unpackedExt *openrtb_ext.ExtRequest) (json.RawMessage, error) {
	if len(req.Ext) == 0 || unpackedExt == nil {
		return json.RawMessage(``), nil
	}

	extCopy := *unpackedExt
	extCopy.Prebid.SChains = nil
	return json.Marshal(extCopy)
}

func prepareSource(req *openrtb.BidRequest, bidder string, sChainsByBidder map[string]*openrtb_ext.ExtRequestPrebidSChainSChain) {
	const sChainWildCard = "*"
	var selectedSChain *openrtb_ext.ExtRequestPrebidSChainSChain

	wildCardSChain := sChainsByBidder[sChainWildCard]
	bidderSChain := sChainsByBidder[bidder]

	// source should not be modified
	if bidderSChain == nil && wildCardSChain == nil {
		return
	}

	if bidderSChain != nil {
		selectedSChain = bidderSChain
	} else {
		selectedSChain = wildCardSChain
	}

	// set source
	if req.Source == nil {
		req.Source = &openrtb.Source{}
	}
	schain := openrtb_ext.ExtRequestPrebidSChain{
		SChain: *selectedSChain,
	}
	sourceExt, err := json.Marshal(schain)
	if err == nil {
		req.Source.Ext = sourceExt
	}
}

// extractBuyerUIDs parses the values from user.ext.prebid.buyeruids, and then deletes those values from the ext.
// This prevents a Bidder from using these values to figure out who else is involved in the Auction.
func extractBuyerUIDs(user *openrtb.User) (map[string]string, error) {
	if user == nil {
		return nil, nil
	}
	if len(user.Ext) == 0 {
		return nil, nil
	}

	var userExt openrtb_ext.ExtUser
	if err := json.Unmarshal(user.Ext, &userExt); err != nil {
		return nil, err
	}
	if userExt.Prebid == nil {
		return nil, nil
	}

	// The API guarantees that user.ext.prebid.buyeruids exists and has at least one ID defined,
	// as long as user.ext.prebid exists.
	buyerUIDs := userExt.Prebid.BuyerUIDs
	userExt.Prebid = nil

	// Remarshal (instead of removing) if the ext has other known fields
	if userExt.Consent != "" || userExt.DigiTrust != nil || len(userExt.Eids) > 0 {
		if newUserExtBytes, err := json.Marshal(userExt); err != nil {
			return nil, err
		} else {
			user.Ext = newUserExtBytes
		}
	} else {
		user.Ext = nil
	}
	return buyerUIDs, nil
}

// splitImps takes a list of Imps and returns a map of imps which have been sanitized for each bidder.
//
// For example, suppose imps has two elements. One goes to rubicon, while the other goes to appnexus and index.
// The returned map will have three keys: rubicon, appnexus, and index--each with one Imp.
// The "imp.ext" value of the appnexus Imp will only contain the "prebid" values, and "appnexus" value at the "bidder" key.
// The "imp.ext" value of the rubicon Imp will only contain the "prebid" values, and "rubicon" value at the "bidder" key.
//
// The goal here is so that Bidders only get Imps and Imp.Ext values which are intended for them.
func splitImps(imps []openrtb.Imp) (map[string][]openrtb.Imp, []error) {
	impExts, err := parseImpExts(imps)
	if err != nil {
		return nil, []error{err}
	}

	splitImps := make(map[string][]openrtb.Imp, len(imps))
	var errList []error

	for i := 0; i < len(imps); i++ {
		imp := imps[i]
		impExt := impExts[i]

		var firstPartyDataContext json.RawMessage
		if context, exists := impExt[openrtb_ext.FirstPartyDataContextExtKey]; exists {
			firstPartyDataContext = context
		}

		rawPrebidExt, ok := impExt[openrtb_ext.PrebidExtKey]

		if ok {
			var prebidExt openrtb_ext.ExtImpPrebid

			if err := json.Unmarshal(rawPrebidExt, &prebidExt); err == nil && prebidExt.Bidder != nil {
				if errs := sanitizedImpCopy(&imp, prebidExt.Bidder, rawPrebidExt, firstPartyDataContext, &splitImps); errs != nil {
					errList = append(errList, errs...)
				}

				continue
			}
		}

		if errs := sanitizedImpCopy(&imp, impExt, rawPrebidExt, firstPartyDataContext, &splitImps); errs != nil {
			errList = append(errList, errs...)
		}
	}

	return splitImps, nil
}

// sanitizedImpCopy returns a copy of imp with its ext filtered so that only "prebid", "context", and bidder params exist.
// It will not mutate the input imp.
// This function will write the new imps to the output map passed in
func sanitizedImpCopy(imp *openrtb.Imp,
	bidderExts map[string]json.RawMessage,
	rawPrebidExt json.RawMessage,
	firstPartyDataContext json.RawMessage,
	out *map[string][]openrtb.Imp) []error {

	var prebidExt map[string]json.RawMessage
	var errs []error

	if err := json.Unmarshal(rawPrebidExt, &prebidExt); err == nil {
		// Remove the entire bidder field. We will already have the content we need in bidderExts. We
		// don't want to include other demand partners' bidder params in the sanitized imp.
		if _, hasBidderField := prebidExt["bidder"]; hasBidderField {
			delete(prebidExt, "bidder")

			var err error
			if rawPrebidExt, err = json.Marshal(prebidExt); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for bidder, ext := range bidderExts {
		if bidder == openrtb_ext.PrebidExtKey || bidder == openrtb_ext.FirstPartyDataContextExtKey {
			continue
		}

		impCopy := *imp
		newExt := make(map[string]json.RawMessage, 3)

		newExt["bidder"] = ext

		if rawPrebidExt != nil {
			newExt[openrtb_ext.PrebidExtKey] = rawPrebidExt
		}

		if len(firstPartyDataContext) > 0 {
			newExt[openrtb_ext.FirstPartyDataContextExtKey] = firstPartyDataContext
		}

		rawExt, err := json.Marshal(newExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		impCopy.Ext = rawExt

		otherImps, _ := (*out)[bidder]
		(*out)[bidder] = append(otherImps, impCopy)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// prepareUser changes req.User so that it's ready for the given bidder.
// This *will* mutate the request, but will *not* mutate any objects nested inside it.
//
// In this function, "givenBidder" may or may not be an alias. "coreBidder" must *not* be an alias.
// It returns true if a Cookie User Sync existed, and false otherwise.
func prepareUser(req *openrtb.BidRequest, givenBidder string, coreBidder openrtb_ext.BidderName, explicitBuyerUIDs map[string]string, usersyncs IdFetcher) bool {
	cookieId, hadCookie := usersyncs.GetId(coreBidder)

	if id, ok := explicitBuyerUIDs[givenBidder]; ok {
		req.User = copyWithBuyerUID(req.User, id)
	} else if hadCookie {
		req.User = copyWithBuyerUID(req.User, cookieId)
	}

	return hadCookie
}

// copyWithBuyerUID either overwrites the BuyerUID property on user with the argument, or returns
// a new (empty) User with the BuyerUID already set.
func copyWithBuyerUID(user *openrtb.User, buyerUID string) *openrtb.User {
	if user == nil {
		return &openrtb.User{
			BuyerUID: buyerUID,
		}
	}
	if user.BuyerUID == "" {
		clone := *user
		clone.BuyerUID = buyerUID
		return &clone
	}
	return user
}

// removeUnpermissionedEids modifies the request to remove any request.user.ext.eids not permissions for the specific bidder
func removeUnpermissionedEids(request *openrtb.BidRequest, bidder string, requestExt *openrtb_ext.ExtRequest) error {
	// ensure request might have eids (as much as we can check before unmarshalling)
	if request.User == nil || len(request.User.Ext) == 0 {
		return nil
	}

	// ensure request has eid permissions to enforce
	if requestExt == nil || requestExt.Prebid.Data == nil || len(requestExt.Prebid.Data.EidPermissions) == 0 {
		return nil
	}

	// low level unmarshal to preserve other request.user.ext values. prebid server is non-destructive.
	var userExt map[string]json.RawMessage
	if err := json.Unmarshal(request.User.Ext, &userExt); err != nil {
		return err
	}

	eidsJSON, eidsSpecified := userExt["eids"]
	if !eidsSpecified {
		return nil
	}

	var eids []openrtb_ext.ExtUserEid
	if err := json.Unmarshal(eidsJSON, &eids); err != nil {
		return err
	}

	// exit early if there are no eids (empty array)
	if len(eids) == 0 {
		return nil
	}

	// translate eid permissions to a map for quick lookup
	eidRules := make(map[string][]string)
	for _, p := range requestExt.Prebid.Data.EidPermissions {
		eidRules[p.Source] = p.Bidders
	}

	eidsAllowed := make([]openrtb_ext.ExtUserEid, 0, len(eids))
	for _, eid := range eids {
		allowed := false
		if rule, hasRule := eidRules[eid.Source]; hasRule {
			for _, ruleBidder := range rule {
				if ruleBidder == "*" || ruleBidder == bidder {
					allowed = true
					break
				}
			}
		} else {
			allowed = true
		}

		if allowed {
			eidsAllowed = append(eidsAllowed, eid)
		}
	}

	// exit early if all eids are allowed and nothing needs to be removed
	if len(eids) == len(eidsAllowed) {
		return nil
	}

	// marshal eidsAllowed back to userExt
	if len(eidsAllowed) == 0 {
		delete(userExt, "eids")
	} else {
		eidsRaw, err := json.Marshal(eidsAllowed)
		if err != nil {
			return err
		}
		userExt["eids"] = eidsRaw
	}

	// exit early if userExt is empty
	if len(userExt) == 0 {
		setUserExtWithCopy(request, nil)
		return nil
	}

	userExtJSON, err := json.Marshal(userExt)
	if err != nil {
		return err
	}
	setUserExtWithCopy(request, userExtJSON)
	return nil
}

func setUserExtWithCopy(request *openrtb.BidRequest, userExtJSON json.RawMessage) {
	userCopy := *request.User
	userCopy.Ext = userExtJSON
	request.User = &userCopy
}

// resolveBidder returns the known BidderName associated with bidder, if bidder is an alias. If it's not an alias, the bidder is returned.
func resolveBidder(bidder string, aliases map[string]string) openrtb_ext.BidderName {
	if coreBidder, ok := aliases[bidder]; ok {
		return openrtb_ext.BidderName(coreBidder)
	}
	return openrtb_ext.BidderName(bidder)
}

// parseImpExts does a partial-unmarshal of the imp[].Ext field.
// The keys in the returned map are expected to be "prebid", "context", CoreBidderNames, or Aliases for this request.
func parseImpExts(imps []openrtb.Imp) ([]map[string]json.RawMessage, error) {
	exts := make([]map[string]json.RawMessage, len(imps))
	// Loop over every impression in the request
	for i := 0; i < len(imps); i++ {
		// Unpack each set of extensions found in the Imp array
		err := json.Unmarshal(imps[i].Ext, &exts[i])
		if err != nil {
			return nil, fmt.Errorf("Error unpacking extensions for Imp[%d]: %s", i, err.Error())
		}
	}
	return exts, nil
}

// parseAliases parses the aliases from the BidRequest
func parseAliases(orig *openrtb.BidRequest) (map[string]string, []error) {
	var aliases map[string]string
	if value, dataType, _, err := jsonparser.Get(orig.Ext, openrtb_ext.PrebidExtKey, "aliases"); dataType == jsonparser.Object && err == nil {
		if err := json.Unmarshal(value, &aliases); err != nil {
			return nil, []error{err}
		}
	} else if dataType != jsonparser.NotExist && err != jsonparser.KeyPathNotFoundError {
		return nil, []error{err}
	}
	return aliases, nil
}

func GetValidBidders(aliases map[string]string) map[string]struct{} {
	validBidders := openrtb_ext.BuildBidderNameHashSet()

	for k := range aliases {
		validBidders[k] = struct{}{}
	}

	return validBidders
}

// Quick little randomizer for a list of strings. Stuffing it in utils to keep other files clean
func randomizeList(list []openrtb_ext.BidderName) {
	l := len(list)
	perm := rand.Perm(l)
	var j int
	for i := 0; i < l; i++ {
		j = perm[i]
		list[i], list[j] = list[j], list[i]
	}
}

func extractBidRequestExt(bidRequest *openrtb.BidRequest) (*openrtb_ext.ExtRequest, error) {
	requestExt := &openrtb_ext.ExtRequest{}

	if bidRequest == nil {
		return requestExt, fmt.Errorf("Error bidRequest should not be nil")
	}

	if len(bidRequest.Ext) > 0 {
		err := json.Unmarshal(bidRequest.Ext, &requestExt)
		if err != nil {
			return requestExt, fmt.Errorf("Error decoding Request.ext : %s", err.Error())
		}
	}
	return requestExt, nil
}

func getExtCacheInstructions(requestExt *openrtb_ext.ExtRequest) extCacheInstructions {
	//returnCreative defaults to true
	cacheInstructions := extCacheInstructions{returnCreative: true}
	foundBidsRC := false
	foundVastRC := false

	if requestExt != nil && requestExt.Prebid.Cache != nil {
		if requestExt.Prebid.Cache.Bids != nil {
			cacheInstructions.cacheBids = true
			if requestExt.Prebid.Cache.Bids.ReturnCreative != nil {
				cacheInstructions.returnCreative = *requestExt.Prebid.Cache.Bids.ReturnCreative
				foundBidsRC = true
			}
		}
		if requestExt.Prebid.Cache.VastXML != nil {
			cacheInstructions.cacheVAST = true
			if requestExt.Prebid.Cache.VastXML.ReturnCreative != nil {
				cacheInstructions.returnCreative = *requestExt.Prebid.Cache.VastXML.ReturnCreative
				foundVastRC = true
			}
		}
	}

	if foundBidsRC && foundVastRC {
		cacheInstructions.returnCreative = *requestExt.Prebid.Cache.Bids.ReturnCreative || *requestExt.Prebid.Cache.VastXML.ReturnCreative
	}

	return cacheInstructions
}

func getExtTargetData(requestExt *openrtb_ext.ExtRequest, cacheInstructions *extCacheInstructions) *targetData {
	var targData *targetData

	if requestExt != nil && requestExt.Prebid.Targeting != nil {
		targData = &targetData{
			priceGranularity:  requestExt.Prebid.Targeting.PriceGranularity,
			includeWinners:    requestExt.Prebid.Targeting.IncludeWinners,
			includeBidderKeys: requestExt.Prebid.Targeting.IncludeBidderKeys,
			includeCacheBids:  cacheInstructions.cacheBids,
			includeCacheVast:  cacheInstructions.cacheVAST,
			includeFormat:     requestExt.Prebid.Targeting.IncludeFormat,
			preferDeals:       requestExt.Prebid.Targeting.PreferDeals,
		}
	}
	return targData
}

func getDebugInfo(bidRequest *openrtb.BidRequest, requestExt *openrtb_ext.ExtRequest) bool {
	return (bidRequest != nil && bidRequest.Test == 1) || (requestExt != nil && requestExt.Prebid.Debug)
}

func getExtBidAdjustmentFactors(requestExt *openrtb_ext.ExtRequest) map[string]float64 {
	var bidAdjustmentFactors map[string]float64
	if requestExt != nil {
		bidAdjustmentFactors = requestExt.Prebid.BidAdjustmentFactors
	}
	return bidAdjustmentFactors
}
