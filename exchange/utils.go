package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/prebid/prebid-server/stored_responses"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/go-gdpr/vendorconsent"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/firstpartydata"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/lmt"
	"github.com/prebid/prebid-server/schain"
)

var integrationTypeMap = map[metrics.RequestType]config.IntegrationType{
	metrics.ReqTypeAMP:      config.IntegrationTypeAMP,
	metrics.ReqTypeORTB2App: config.IntegrationTypeApp,
	metrics.ReqTypeVideo:    config.IntegrationTypeVideo,
	metrics.ReqTypeORTB2Web: config.IntegrationTypeWeb,
}

const unknownBidder string = ""

// cleanOpenRTBRequests splits the input request into requests which are sanitized for each bidder. Intended behavior is:
//
//   1. BidRequest.Imp[].Ext will only contain the "prebid" field and a "bidder" field which has the params for the intended Bidder.
//   2. Every BidRequest.Imp[] requested Bids from the Bidder who keys it.
//   3. BidRequest.User.BuyerUID will be set to that Bidder's ID.
func cleanOpenRTBRequests(ctx context.Context,
	auctionReq AuctionRequest,
	requestExt *openrtb_ext.ExtRequest,
	bidderToSyncerKey map[string]string,
	metricsEngine metrics.MetricsEngine,
	gdprDefaultValue gdpr.Signal,
	gdprPerms gdpr.Permissions,
	privacyConfig config.Privacy,
	tcf2Cfg gdpr.TCF2ConfigReader,
) (allowedBidderRequests []BidderRequest, privacyLabels metrics.PrivacyLabels, errs []error) {

	req := auctionReq.BidRequestWrapper
	aliases, errs := parseAliases(req.BidRequest)
	if len(errs) > 0 {
		return
	}

	allowedBidderRequests = make([]BidderRequest, 0, 0)

	bidderImpWithBidResp := stored_responses.InitStoredBidResponses(req.BidRequest, auctionReq.StoredBidResponses)

	impsByBidder, err := splitImps(req.BidRequest.Imp)
	if err != nil {
		errs = []error{err}
		return
	}

	aliasesGVLIDs, errs := parseAliasesGVLIDs(req.BidRequest)
	if len(errs) > 0 {
		return
	}

	var allBidderRequests []BidderRequest
	allBidderRequests, errs = getAuctionBidderRequests(auctionReq, requestExt, bidderToSyncerKey, impsByBidder, aliases)

	bidderNameToBidderReq := buildBidResponseRequest(req.BidRequest, bidderImpWithBidResp, aliases)
	//this function should be executed after getAuctionBidderRequests
	allBidderRequests = mergeBidderRequests(allBidderRequests, bidderNameToBidderReq)

	gdprSignal, err := extractGDPR(req.BidRequest)
	if err != nil {
		errs = append(errs, err)
	}
	consent, err := extractConsent(req.BidRequest)
	if err != nil {
		errs = append(errs, err)
	}
	gdprApplies := gdprSignal == gdpr.SignalYes || (gdprSignal == gdpr.SignalAmbiguous && gdprDefaultValue == gdpr.SignalYes)

	ccpaEnforcer, err := extractCCPA(req.BidRequest, privacyConfig, &auctionReq.Account, aliases, integrationTypeMap[auctionReq.LegacyLabels.RType])
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

	var gdprEnforced bool
	if gdprApplies {
		gdprEnforced = tcf2Cfg.IntegrationEnabled(integrationTypeMap[auctionReq.LegacyLabels.RType])
	}

	if gdprEnforced {
		privacyLabels.GDPREnforced = true
		parsedConsent, err := vendorconsent.ParseString(consent)
		if err == nil {
			version := int(parsedConsent.Version())
			privacyLabels.GDPRTCFVersion = metrics.TCFVersionToValue(version)
		}
	}

	// bidder level privacy policies
	for _, bidderRequest := range allBidderRequests {
		bidRequestAllowed := true

		// CCPA
		privacyEnforcement.CCPA = ccpaEnforcer.ShouldEnforce(bidderRequest.BidderName.String())

		// GDPR
		if gdprEnforced {
			var publisherID = auctionReq.LegacyLabels.PubID
			auctionPermissions, err := gdprPerms.AuctionActivitiesAllowed(ctx, bidderRequest.BidderCoreName, bidderRequest.BidderName, publisherID, gdprSignal, consent, aliasesGVLIDs)
			bidRequestAllowed = auctionPermissions.AllowBidRequest

			if err == nil {
				privacyEnforcement.GDPRGeo = !auctionPermissions.PassGeo
				privacyEnforcement.GDPRID = !auctionPermissions.PassID
			} else {
				privacyEnforcement.GDPRGeo = true
				privacyEnforcement.GDPRID = true
			}

			if !bidRequestAllowed {
				metricsEngine.RecordAdapterGDPRRequestBlocked(bidderRequest.BidderCoreName)
			}
		}

		if auctionReq.FirstPartyData != nil && auctionReq.FirstPartyData[bidderRequest.BidderName] != nil {
			applyFPD(auctionReq.FirstPartyData[bidderRequest.BidderName], bidderRequest.BidRequest)
		}

		if bidRequestAllowed {
			privacyEnforcement.Apply(bidderRequest.BidRequest)

			allowedBidderRequests = append(allowedBidderRequests, bidderRequest)
		}
	}

	return
}

func ccpaEnabled(account *config.Account, privacyConfig config.Privacy, requestType config.IntegrationType) bool {
	if accountEnabled := account.CCPA.EnabledForIntegrationType(requestType); accountEnabled != nil {
		return *accountEnabled
	}
	return privacyConfig.CCPA.Enforce
}

func extractCCPA(orig *openrtb2.BidRequest, privacyConfig config.Privacy, account *config.Account, aliases map[string]string, requestType config.IntegrationType) (privacy.PolicyEnforcer, error) {
	// Quick extra wrapper until RequestWrapper makes its way into CleanRequests
	ccpaPolicy, err := ccpa.ReadFromRequestWrapper(&openrtb_ext.RequestWrapper{BidRequest: orig})
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

func extractLMT(orig *openrtb2.BidRequest, privacyConfig config.Privacy) privacy.PolicyEnforcer {
	return privacy.EnabledPolicyEnforcer{
		Enabled:        privacyConfig.LMT.Enforce,
		PolicyEnforcer: lmt.ReadFromRequest(orig),
	}
}

func getAuctionBidderRequests(auctionRequest AuctionRequest,
	requestExt *openrtb_ext.ExtRequest,
	bidderToSyncerKey map[string]string,
	impsByBidder map[string][]openrtb2.Imp,
	aliases map[string]string) ([]BidderRequest, []error) {

	bidderRequests := make([]BidderRequest, 0, len(impsByBidder))
	req := auctionRequest.BidRequestWrapper
	explicitBuyerUIDs, err := extractBuyerUIDs(req.BidRequest.User)
	if err != nil {
		return nil, []error{err}
	}

	bidderParamsInReqExt, err := adapters.ExtractReqExtBidderParamsEmbeddedMap(req.BidRequest)
	if err != nil {
		return nil, []error{err}
	}

	sChainWriter, err := schain.NewSChainWriter(requestExt)
	if err != nil {
		return nil, []error{err}
	}

	var errs []error
	for bidder, imps := range impsByBidder {
		coreBidder := resolveBidder(bidder, aliases)

		reqCopy := *req.BidRequest
		reqCopy.Imp = imps

		sChainWriter.Write(&reqCopy, bidder)

		if len(bidderParamsInReqExt) != 0 {

			// Update bidder-params(requestExt.Prebid.BidderParams) for the bidder to only contain bidder-params for
			// this bidder only and remove bidder-params for all other bidders from requestExt.Prebid.BidderParams
			params, err := getBidderParamsForBidder(bidderParamsInReqExt, bidder)
			if err != nil {
				return nil, []error{err}
			}

			requestExt.Prebid.BidderParams = params
		}

		reqExt, err := getExtJson(req.BidRequest, requestExt)
		if err != nil {
			return nil, []error{err}
		}
		reqCopy.Ext = reqExt

		if err := removeUnpermissionedEids(&reqCopy, bidder, requestExt); err != nil {
			errs = append(errs, fmt.Errorf("unable to enforce request.ext.prebid.data.eidpermissions because %v", err))
			continue
		}

		bidderRequest := BidderRequest{
			BidderName:     openrtb_ext.BidderName(bidder),
			BidderCoreName: coreBidder,
			BidRequest:     &reqCopy,
			BidderLabels: metrics.AdapterLabels{
				Source:      auctionRequest.LegacyLabels.Source,
				RType:       auctionRequest.LegacyLabels.RType,
				Adapter:     coreBidder,
				PubID:       auctionRequest.LegacyLabels.PubID,
				CookieFlag:  auctionRequest.LegacyLabels.CookieFlag,
				AdapterBids: metrics.AdapterBidPresent,
			},
		}

		syncerKey := bidderToSyncerKey[string(coreBidder)]
		if hadSync := prepareUser(&reqCopy, bidder, syncerKey, explicitBuyerUIDs, auctionRequest.UserSyncs); !hadSync && req.BidRequest.App == nil {
			bidderRequest.BidderLabels.CookieFlag = metrics.CookieFlagNo
		} else {
			bidderRequest.BidderLabels.CookieFlag = metrics.CookieFlagYes
		}

		bidderRequests = append(bidderRequests, bidderRequest)
	}
	return bidderRequests, errs
}

func getBidderParamsForBidder(bidderParamsInReqExt map[string]map[string]json.RawMessage, bidder string) (json.RawMessage, error) {
	var params json.RawMessage
	if bidderParams, ok := bidderParamsInReqExt[bidder]; ok {
		var err error
		params, err = json.Marshal(bidderParams)
		if err != nil {
			return nil, err
		}
	}
	return params, nil
}

func getExtJson(req *openrtb2.BidRequest, unpackedExt *openrtb_ext.ExtRequest) (json.RawMessage, error) {
	if len(req.Ext) == 0 || unpackedExt == nil {
		return json.RawMessage(``), nil
	}

	extCopy := *unpackedExt
	extCopy.Prebid.SChains = nil
	return json.Marshal(extCopy)
}

// extractBuyerUIDs parses the values from user.ext.prebid.buyeruids, and then deletes those values from the ext.
// This prevents a Bidder from using these values to figure out who else is involved in the Auction.
func extractBuyerUIDs(user *openrtb2.User) (map[string]string, error) {
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
	if userExt.Consent != "" || len(userExt.Eids) > 0 {
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
func splitImps(imps []openrtb2.Imp) (map[string][]openrtb2.Imp, error) {
	bidderImps := make(map[string][]openrtb2.Imp)

	for i, imp := range imps {
		var impExt map[string]json.RawMessage
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			return nil, fmt.Errorf("invalid json for imp[%d]: %v", i, err)
		}

		var impExtPrebid map[string]json.RawMessage
		if impExtPrebidJSON, exists := impExt[openrtb_ext.PrebidExtKey]; exists {
			// validation already performed by impExt unmarshal. no error is possible here, proven by tests.
			json.Unmarshal(impExtPrebidJSON, &impExtPrebid)
		}

		var impExtPrebidBidder map[string]json.RawMessage
		if impExtPrebidBidderJSON, exists := impExtPrebid[openrtb_ext.PrebidExtBidderKey]; exists {
			// validation already performed by impExt unmarshal. no error is possible here, proven by tests.
			json.Unmarshal(impExtPrebidBidderJSON, &impExtPrebidBidder)
		}

		sanitizedImpExt, err := createSanitizedImpExt(impExt, impExtPrebid)
		if err != nil {
			return nil, fmt.Errorf("unable to remove other bidder fields for imp[%d]: %v", i, err)
		}

		for bidder, bidderExt := range extractBidderExts(impExt, impExtPrebidBidder) {
			impCopy := imp

			sanitizedImpExt[openrtb_ext.PrebidExtBidderKey] = bidderExt

			impExtJSON, err := json.Marshal(sanitizedImpExt)
			if err != nil {
				return nil, fmt.Errorf("unable to remove other bidder fields for imp[%d]: cannot marshal ext: %v", i, err)
			}
			impCopy.Ext = impExtJSON

			bidderImps[bidder] = append(bidderImps[bidder], impCopy)
		}
	}

	return bidderImps, nil
}

func createSanitizedImpExt(impExt, impExtPrebid map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	sanitizedImpExt := make(map[string]json.RawMessage, 3)

	delete(impExtPrebid, openrtb_ext.PrebidExtBidderKey)
	if len(impExtPrebid) > 0 {
		if impExtPrebidJSON, err := json.Marshal(impExtPrebid); err == nil {
			sanitizedImpExt[openrtb_ext.PrebidExtKey] = impExtPrebidJSON
		} else {
			return nil, fmt.Errorf("cannot marshal ext.prebid: %v", err)
		}
	}

	if v, exists := impExt[openrtb_ext.FirstPartyDataExtKey]; exists {
		sanitizedImpExt[openrtb_ext.FirstPartyDataExtKey] = v
	}

	if v, exists := impExt[openrtb_ext.FirstPartyDataContextExtKey]; exists {
		sanitizedImpExt[openrtb_ext.FirstPartyDataContextExtKey] = v
	}

	if v, exists := impExt[openrtb_ext.SKAdNExtKey]; exists {
		sanitizedImpExt[openrtb_ext.SKAdNExtKey] = v
	}

	if v, exists := impExt[string(openrtb_ext.GPIDKey)]; exists {
		sanitizedImpExt[openrtb_ext.GPIDKey] = v
	}

	return sanitizedImpExt, nil
}

func extractBidderExts(impExt, impExtPrebidBidders map[string]json.RawMessage) map[string]json.RawMessage {
	bidderExts := make(map[string]json.RawMessage)

	// prefer imp.ext.prebid.bidder.BIDDER
	for bidder, bidderExt := range impExtPrebidBidders {
		bidderExts[bidder] = bidderExt
	}

	// fallback to imp.BIDDER
	for bidder, bidderExt := range impExt {
		if isSpecialField(bidder) {
			continue
		}

		if _, exists := bidderExts[bidder]; !exists {
			bidderExts[bidder] = bidderExt
		}
	}

	return bidderExts
}

func isSpecialField(bidder string) bool {
	return bidder == openrtb_ext.FirstPartyDataContextExtKey ||
		bidder == openrtb_ext.FirstPartyDataExtKey ||
		bidder == openrtb_ext.SKAdNExtKey ||
		bidder == openrtb_ext.GPIDKey ||
		bidder == openrtb_ext.PrebidExtKey
}

// prepareUser changes req.User so that it's ready for the given bidder.
// This *will* mutate the request, but will *not* mutate any objects nested inside it.
//
// In this function, "givenBidder" may or may not be an alias. "coreBidder" must *not* be an alias.
// It returns true if a Cookie User Sync existed, and false otherwise.
func prepareUser(req *openrtb2.BidRequest, givenBidder, syncerKey string, explicitBuyerUIDs map[string]string, usersyncs IdFetcher) bool {
	cookieId, hadCookie, _ := usersyncs.GetUID(syncerKey)

	if id, ok := explicitBuyerUIDs[givenBidder]; ok {
		req.User = copyWithBuyerUID(req.User, id)
	} else if hadCookie {
		req.User = copyWithBuyerUID(req.User, cookieId)
	}

	return hadCookie
}

// copyWithBuyerUID either overwrites the BuyerUID property on user with the argument, or returns
// a new (empty) User with the BuyerUID already set.
func copyWithBuyerUID(user *openrtb2.User, buyerUID string) *openrtb2.User {
	if user == nil {
		return &openrtb2.User{
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
func removeUnpermissionedEids(request *openrtb2.BidRequest, bidder string, requestExt *openrtb_ext.ExtRequest) error {
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

func setUserExtWithCopy(request *openrtb2.BidRequest, userExtJSON json.RawMessage) {
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

// parseAliases parses the aliases from the BidRequest
func parseAliases(orig *openrtb2.BidRequest) (map[string]string, []error) {
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

// parseAliasesGVLIDs parses the Bidder Alias GVLIDs from the BidRequest
func parseAliasesGVLIDs(orig *openrtb2.BidRequest) (map[string]uint16, []error) {
	var aliasesGVLIDs map[string]uint16
	if value, dataType, _, err := jsonparser.Get(orig.Ext, openrtb_ext.PrebidExtKey, "aliasgvlids"); dataType == jsonparser.Object && err == nil {
		if err := json.Unmarshal(value, &aliasesGVLIDs); err != nil {
			return nil, []error{err}
		}
	} else if dataType != jsonparser.NotExist && err != jsonparser.KeyPathNotFoundError {
		return nil, []error{err}
	}
	return aliasesGVLIDs, nil
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

func extractBidRequestExt(bidRequest *openrtb2.BidRequest) (*openrtb_ext.ExtRequest, error) {
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

// getDebugInfo returns the boolean flags that allow for debug information in bidResponse.Ext, the SeatBid.httpcalls slice, and
// also sets the debugLog information
func getDebugInfo(bidRequest *openrtb2.BidRequest, requestExt *openrtb_ext.ExtRequest, accountDebugFlag bool, debugLog *DebugLog) (bool, bool, *DebugLog) {
	requestDebugAllow := parseRequestDebugValues(bidRequest, requestExt)
	debugLog = setDebugLogValues(accountDebugFlag, debugLog)

	responseDebugAllow := (requestDebugAllow && accountDebugFlag) || debugLog.DebugEnabledOrOverridden
	accountDebugAllow := (requestDebugAllow && accountDebugFlag) || (debugLog.DebugEnabledOrOverridden && accountDebugFlag)

	return responseDebugAllow, accountDebugAllow, debugLog
}

// setDebugLogValues initializes the DebugLog if nil. It also sets the value of the debugInfo flag
// used in HoldAuction
func setDebugLogValues(accountDebugFlag bool, debugLog *DebugLog) *DebugLog {
	if debugLog == nil {
		debugLog = &DebugLog{}
	}

	debugLog.Enabled = debugLog.DebugEnabledOrOverridden || accountDebugFlag
	return debugLog
}

func parseRequestDebugValues(bidRequest *openrtb2.BidRequest, requestExt *openrtb_ext.ExtRequest) bool {
	return (bidRequest != nil && bidRequest.Test == 1) || (requestExt != nil && requestExt.Prebid.Debug)
}

func getExtBidAdjustmentFactors(requestExt *openrtb_ext.ExtRequest) map[string]float64 {
	var bidAdjustmentFactors map[string]float64
	if requestExt != nil {
		bidAdjustmentFactors = requestExt.Prebid.BidAdjustmentFactors
	}
	return bidAdjustmentFactors
}

func applyFPD(fpd *firstpartydata.ResolvedFirstPartyData, bidReq *openrtb2.BidRequest) {
	if fpd.Site != nil {
		bidReq.Site = fpd.Site
	}
	if fpd.App != nil {
		bidReq.App = fpd.App
	}
	if fpd.User != nil {
		bidReq.User = fpd.User
	}
}

func buildBidResponseRequest(req *openrtb2.BidRequest,
	bidderImpResponses stored_responses.BidderImpsWithBidResponses,
	aliases map[string]string) map[openrtb_ext.BidderName]BidderRequest {
	bidderToBidderResponse := make(map[openrtb_ext.BidderName]BidderRequest)
	for bidderName, impResps := range bidderImpResponses {
		resolvedBidder := resolveBidder(string(bidderName), aliases)
		bidderToBidderResponse[bidderName] = BidderRequest{
			BidRequest:            req,
			BidderCoreName:        resolvedBidder,
			BidderName:            bidderName,
			BidderStoredResponses: impResps,
			BidderLabels:          metrics.AdapterLabels{Adapter: resolvedBidder},
		}
	}
	return bidderToBidderResponse
}

func mergeBidderRequests(allBidderRequests []BidderRequest, bidderNameToBidderReq map[openrtb_ext.BidderName]BidderRequest) []BidderRequest {
	if len(allBidderRequests) == 0 && len(bidderNameToBidderReq) == 0 {
		return allBidderRequests
	}
	if len(allBidderRequests) == 0 && len(bidderNameToBidderReq) > 0 {
		for _, v := range bidderNameToBidderReq {
			allBidderRequests = append(allBidderRequests, v)
		}
		return allBidderRequests
	} else if len(allBidderRequests) > 0 && len(bidderNameToBidderReq) > 0 {
		//merge bidder requests with real imps and imps with stored resp
		for bn, br := range bidderNameToBidderReq {
			found := false
			for i, ar := range allBidderRequests {
				if ar.BidderName == bn {
					//bidder req with real imps and imps with stored resp
					allBidderRequests[i].BidderStoredResponses = br.BidderStoredResponses
					found = true
					break
				}
			}
			if !found {
				//bidder req with stored bid responses only
				br.BidRequest.Imp = nil // to indicate this bidder request has bidder responses only
				allBidderRequests = append(allBidderRequests, br)
			}
		}
	}
	return allBidderRequests
}
