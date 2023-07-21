package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"

	"github.com/buger/jsonparser"
	"github.com/prebid/go-gdpr/vendorconsent"
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/firstpartydata"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/lmt"
	"github.com/prebid/prebid-server/schain"
	"github.com/prebid/prebid-server/stored_responses"
	"github.com/prebid/prebid-server/util/ptrutil"
)

var channelTypeMap = map[metrics.RequestType]config.ChannelType{
	metrics.ReqTypeAMP:      config.ChannelAMP,
	metrics.ReqTypeORTB2App: config.ChannelApp,
	metrics.ReqTypeVideo:    config.ChannelVideo,
	metrics.ReqTypeORTB2Web: config.ChannelWeb,
}

const unknownBidder string = ""

type requestSplitter struct {
	bidderToSyncerKey map[string]string
	me                metrics.MetricsEngine
	privacyConfig     config.Privacy
	gdprPermsBuilder  gdpr.PermissionsBuilder
	hostSChainNode    *openrtb2.SupplyChainNode
	bidderInfo        config.BidderInfos
}

// cleanOpenRTBRequests splits the input request into requests which are sanitized for each bidder. Intended behavior is:
//
//  1. BidRequest.Imp[].Ext will only contain the "prebid" field and a "bidder" field which has the params for the intended Bidder.
//  2. Every BidRequest.Imp[] requested Bids from the Bidder who keys it.
//  3. BidRequest.User.BuyerUID will be set to that Bidder's ID.
func (rs *requestSplitter) cleanOpenRTBRequests(ctx context.Context,
	auctionReq AuctionRequest,
	requestExt *openrtb_ext.ExtRequest,
	gdprDefaultValue gdpr.Signal,
) (allowedBidderRequests []BidderRequest, privacyLabels metrics.PrivacyLabels, errs []error) {

	req := auctionReq.BidRequestWrapper
	aliases, errs := parseAliases(req.BidRequest)
	if len(errs) > 0 {
		return
	}

	allowedBidderRequests = make([]BidderRequest, 0)

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
	allBidderRequests, errs = getAuctionBidderRequests(auctionReq, requestExt, rs.bidderToSyncerKey, impsByBidder, aliases, rs.hostSChainNode)

	bidderNameToBidderReq := buildBidResponseRequest(req.BidRequest, bidderImpWithBidResp, aliases, auctionReq.BidderImpReplaceImpID)
	//this function should be executed after getAuctionBidderRequests
	allBidderRequests = mergeBidderRequests(allBidderRequests, bidderNameToBidderReq)

	var gpp gpplib.GppContainer
	if req.BidRequest.Regs != nil && len(req.BidRequest.Regs.GPP) > 0 {
		gpp, err = gpplib.Parse(req.BidRequest.Regs.GPP)
		if err != nil {
			errs = append(errs, err)
		}
	}

	gdprSignal, err := getGDPR(req)
	if err != nil {
		errs = append(errs, err)
	}

	consent, err := getConsent(req, gpp)
	if err != nil {
		errs = append(errs, err)
	}
	gdprApplies := gdprSignal == gdpr.SignalYes || (gdprSignal == gdpr.SignalAmbiguous && gdprDefaultValue == gdpr.SignalYes)

	ccpaEnforcer, err := extractCCPA(req.BidRequest, rs.privacyConfig, &auctionReq.Account, aliases, channelTypeMap[auctionReq.LegacyLabels.RType], gpp)
	if err != nil {
		errs = append(errs, err)
	}

	lmtEnforcer := extractLMT(req.BidRequest, rs.privacyConfig)

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
	var gdprPerms gdpr.Permissions = &gdpr.AlwaysAllow{}

	if gdprApplies {
		gdprEnforced = auctionReq.TCF2Config.ChannelEnabled(channelTypeMap[auctionReq.LegacyLabels.RType])
	}

	if gdprEnforced {
		privacyLabels.GDPREnforced = true
		parsedConsent, err := vendorconsent.ParseString(consent)
		if err == nil {
			version := int(parsedConsent.Version())
			privacyLabels.GDPRTCFVersion = metrics.TCFVersionToValue(version)
		}

		gdprRequestInfo := gdpr.RequestInfo{
			AliasGVLIDs: aliasesGVLIDs,
			Consent:     consent,
			GDPRSignal:  gdprSignal,
			PublisherID: auctionReq.LegacyLabels.PubID,
		}
		gdprPerms = rs.gdprPermsBuilder(auctionReq.TCF2Config, gdprRequestInfo)
	}

	// bidder level privacy policies
	for _, bidderRequest := range allBidderRequests {
		bidRequestAllowed := true

		// fetchBids activity
		fetchBidsActivityAllowed := auctionReq.Activities.Allow(privacy.ActivityFetchBids,
			privacy.ScopedName{Scope: privacy.ScopeTypeBidder, Name: bidderRequest.BidderName.String()})
		if fetchBidsActivityAllowed == privacy.ActivityDeny {
			// skip the call to a bidder if fetchBids activity is not allowed
			// do not add this bidder to allowedBidderRequests
			continue
		}

		// CCPA
		privacyEnforcement.CCPA = ccpaEnforcer.ShouldEnforce(bidderRequest.BidderName.String())

		// GDPR
		if gdprEnforced {
			auctionPermissions, err := gdprPerms.AuctionActivitiesAllowed(ctx, bidderRequest.BidderCoreName, bidderRequest.BidderName)
			bidRequestAllowed = auctionPermissions.AllowBidRequest

			if err == nil {
				privacyEnforcement.GDPRGeo = !auctionPermissions.PassGeo
				privacyEnforcement.GDPRID = !auctionPermissions.PassID
			} else {
				privacyEnforcement.GDPRGeo = true
				privacyEnforcement.GDPRID = true
			}

			if !bidRequestAllowed {
				rs.me.RecordAdapterGDPRRequestBlocked(bidderRequest.BidderCoreName)
			}
		}

		if auctionReq.FirstPartyData != nil && auctionReq.FirstPartyData[bidderRequest.BidderName] != nil {
			applyFPD(auctionReq.FirstPartyData[bidderRequest.BidderName], bidderRequest.BidRequest)
		}

		if bidRequestAllowed {
			privacyEnforcement.Apply(bidderRequest.BidRequest)
			allowedBidderRequests = append(allowedBidderRequests, bidderRequest)
		}
		// GPP downgrade: always downgrade unless we can confirm GPP is supported
		if shouldSetLegacyPrivacy(rs.bidderInfo, string(bidderRequest.BidderCoreName)) {
			setLegacyGDPRFromGPP(bidderRequest.BidRequest, gpp)
			setLegacyUSPFromGPP(bidderRequest.BidRequest, gpp)
		}
	}

	return
}

func shouldSetLegacyPrivacy(bidderInfo config.BidderInfos, bidder string) bool {
	binfo, defined := bidderInfo[bidder]

	if !defined || binfo.OpenRTB == nil {
		return true
	}

	return !binfo.OpenRTB.GPPSupported
}

func ccpaEnabled(account *config.Account, privacyConfig config.Privacy, requestType config.ChannelType) bool {
	if accountEnabled := account.CCPA.EnabledForChannelType(requestType); accountEnabled != nil {
		return *accountEnabled
	}
	return privacyConfig.CCPA.Enforce
}

func extractCCPA(orig *openrtb2.BidRequest, privacyConfig config.Privacy, account *config.Account, aliases map[string]string, requestType config.ChannelType, gpp gpplib.GppContainer) (privacy.PolicyEnforcer, error) {
	// Quick extra wrapper until RequestWrapper makes its way into CleanRequests
	ccpaPolicy, err := ccpa.ReadFromRequestWrapper(&openrtb_ext.RequestWrapper{BidRequest: orig}, gpp)
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

func ExtractReqExtBidderParamsMap(bidRequest *openrtb2.BidRequest) (map[string]json.RawMessage, error) {
	if bidRequest == nil {
		return nil, errors.New("error bidRequest should not be nil")
	}

	reqExt := &openrtb_ext.ExtRequest{}
	if len(bidRequest.Ext) > 0 {
		err := json.Unmarshal(bidRequest.Ext, &reqExt)
		if err != nil {
			return nil, fmt.Errorf("error decoding Request.ext : %s", err.Error())
		}
	}

	if reqExt.Prebid.BidderParams == nil {
		return nil, nil
	}

	var bidderParams map[string]json.RawMessage
	err := json.Unmarshal(reqExt.Prebid.BidderParams, &bidderParams)
	if err != nil {
		return nil, err
	}

	return bidderParams, nil
}

func getAuctionBidderRequests(auctionRequest AuctionRequest,
	requestExt *openrtb_ext.ExtRequest,
	bidderToSyncerKey map[string]string,
	impsByBidder map[string][]openrtb2.Imp,
	aliases map[string]string,
	hostSChainNode *openrtb2.SupplyChainNode) ([]BidderRequest, []error) {

	bidderRequests := make([]BidderRequest, 0, len(impsByBidder))
	req := auctionRequest.BidRequestWrapper
	explicitBuyerUIDs, err := extractBuyerUIDs(req.BidRequest.User)
	if err != nil {
		return nil, []error{err}
	}

	bidderParamsInReqExt, err := ExtractReqExtBidderParamsMap(req.BidRequest)
	if err != nil {
		return nil, []error{err}
	}

	sChainWriter, err := schain.NewSChainWriter(requestExt, hostSChainNode)
	if err != nil {
		return nil, []error{err}
	}

	var errs []error
	for bidder, imps := range impsByBidder {
		coreBidder := resolveBidder(bidder, aliases)

		reqCopy := *req.BidRequest
		reqCopy.Imp = imps

		sChainWriter.Write(&reqCopy, bidder)

		reqCopy.Ext, err = buildRequestExtForBidder(bidder, req.BidRequest.Ext, requestExt, bidderParamsInReqExt, auctionRequest.Account.AlternateBidderCodes)
		if err != nil {
			return nil, []error{err}
		}

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

func buildRequestExtForBidder(bidder string, requestExt json.RawMessage, requestExtParsed *openrtb_ext.ExtRequest, bidderParamsInReqExt map[string]json.RawMessage, cfgABC *openrtb_ext.ExtAlternateBidderCodes) (json.RawMessage, error) {
	// Resolve alternatebiddercode for current bidder
	var reqABC *openrtb_ext.ExtAlternateBidderCodes
	if len(requestExt) != 0 && requestExtParsed != nil && requestExtParsed.Prebid.AlternateBidderCodes != nil {
		reqABC = requestExtParsed.Prebid.AlternateBidderCodes
	}
	alternateBidderCodes := buildRequestExtAlternateBidderCodes(bidder, cfgABC, reqABC)

	if (len(requestExt) == 0 || requestExtParsed == nil) && alternateBidderCodes == nil {
		return nil, nil
	}

	// Resolve Bidder Params
	var bidderParams json.RawMessage
	if bidderParamsInReqExt != nil {
		bidderParams = bidderParamsInReqExt[bidder]
	}

	// Copy Allowed Fields
	// Per: https://docs.prebid.org/prebid-server/endpoints/openrtb2/pbs-endpoint-auction.html#prebid-server-ortb2-extension-summary
	prebid := openrtb_ext.ExtRequestPrebid{
		BidderParams:         bidderParams,
		AlternateBidderCodes: alternateBidderCodes,
	}

	if requestExtParsed != nil {
		prebid.CurrencyConversions = requestExtParsed.Prebid.CurrencyConversions
		prebid.Integration = requestExtParsed.Prebid.Integration
		prebid.Channel = requestExtParsed.Prebid.Channel
		prebid.Debug = requestExtParsed.Prebid.Debug
		prebid.Server = requestExtParsed.Prebid.Server
		prebid.MultiBid = buildRequestExtMultiBid(bidder, requestExtParsed.Prebid.MultiBid, alternateBidderCodes)
	}

	// Marshal New Prebid Object
	prebidJson, err := json.Marshal(prebid)
	if err != nil {
		return nil, err
	}

	// Parse Existing Ext
	extMap := make(map[string]json.RawMessage)
	if len(requestExt) != 0 {
		if err := json.Unmarshal(requestExt, &extMap); err != nil {
			return nil, err
		}
	}

	// Update Ext With Prebid Json
	if bytes.Equal(prebidJson, []byte(`{}`)) {
		delete(extMap, "prebid")
	} else {
		extMap["prebid"] = prebidJson
	}

	if len(extMap) > 0 {
		return json.Marshal(extMap)
	} else {
		return nil, nil
	}
}

func buildRequestExtAlternateBidderCodes(bidder string, accABC *openrtb_ext.ExtAlternateBidderCodes, reqABC *openrtb_ext.ExtAlternateBidderCodes) *openrtb_ext.ExtAlternateBidderCodes {
	if reqABC != nil {
		alternateBidderCodes := &openrtb_ext.ExtAlternateBidderCodes{
			Enabled: reqABC.Enabled,
		}
		if bidderCodes, ok := reqABC.Bidders[bidder]; ok {
			alternateBidderCodes.Bidders = map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
				bidder: bidderCodes,
			}
		}
		return alternateBidderCodes
	}

	if accABC != nil {
		alternateBidderCodes := &openrtb_ext.ExtAlternateBidderCodes{
			Enabled: accABC.Enabled,
		}
		if bidderCodes, ok := accABC.Bidders[bidder]; ok {
			alternateBidderCodes.Bidders = map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
				bidder: bidderCodes,
			}
		}
		return alternateBidderCodes
	}

	return nil
}

func buildRequestExtMultiBid(adapter string, reqMultiBid []*openrtb_ext.ExtMultiBid, adapterABC *openrtb_ext.ExtAlternateBidderCodes) []*openrtb_ext.ExtMultiBid {
	adapterMultiBid := make([]*openrtb_ext.ExtMultiBid, 0)
	for _, multiBid := range reqMultiBid {
		if multiBid.Bidder != "" {
			if multiBid.Bidder == adapter || isBidderInExtAlternateBidderCodes(adapter, multiBid.Bidder, adapterABC) {
				adapterMultiBid = append(adapterMultiBid, multiBid)
			}
		} else {
			for _, bidder := range multiBid.Bidders {
				if bidder == adapter || isBidderInExtAlternateBidderCodes(adapter, bidder, adapterABC) {
					adapterMultiBid = append(adapterMultiBid, &openrtb_ext.ExtMultiBid{
						Bidders: []string{bidder},
						MaxBids: multiBid.MaxBids,
					})
				}
			}
		}
	}

	if len(adapterMultiBid) > 0 {
		return adapterMultiBid
	}

	return nil
}

func isBidderInExtAlternateBidderCodes(adapter, currentMultiBidBidder string, adapterABC *openrtb_ext.ExtAlternateBidderCodes) bool {
	if adapterABC != nil {
		if abc, ok := adapterABC.Bidders[adapter]; ok {
			for _, bidder := range abc.AllowedBidderCodes {
				if bidder == "*" || bidder == currentMultiBidBidder {
					return true
				}
			}
		}
	}
	return false
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

		for bidder, bidderExt := range impExtPrebidBidder {
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

var allowedImpExtFields = map[string]interface{}{
	openrtb_ext.AuctionEnvironmentKey:       struct{}{},
	openrtb_ext.FirstPartyDataExtKey:        struct{}{},
	openrtb_ext.FirstPartyDataContextExtKey: struct{}{},
	openrtb_ext.GPIDKey:                     struct{}{},
	openrtb_ext.SKAdNExtKey:                 struct{}{},
	openrtb_ext.TIDKey:                      struct{}{},
}

var allowedImpExtPrebidFields = map[string]interface{}{
	openrtb_ext.IsRewardedInventoryKey: struct{}{},
	openrtb_ext.OptionsKey:             struct{}{},
}

func createSanitizedImpExt(impExt, impExtPrebid map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	sanitizedImpExt := make(map[string]json.RawMessage, 6)
	sanitizedImpPrebidExt := make(map[string]json.RawMessage, 2)

	// copy allowed imp[].ext.prebid fields
	for k := range allowedImpExtPrebidFields {
		if v, exists := impExtPrebid[k]; exists {
			sanitizedImpPrebidExt[k] = v
		}
	}

	// marshal sanitized imp[].ext.prebid
	if len(sanitizedImpPrebidExt) > 0 {
		if impExtPrebidJSON, err := json.Marshal(sanitizedImpPrebidExt); err == nil {
			sanitizedImpExt[openrtb_ext.PrebidExtKey] = impExtPrebidJSON
		} else {
			return nil, fmt.Errorf("cannot marshal ext.prebid: %v", err)
		}
	}

	// copy reserved imp[].ext fields known to not be bidder names
	for k := range allowedImpExtFields {
		if v, exists := impExt[k]; exists {
			sanitizedImpExt[k] = v
		}
	}

	return sanitizedImpExt, nil
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

	var eids []openrtb2.EID
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

	eidsAllowed := make([]openrtb2.EID, 0, len(eids))
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

func getExtCacheInstructions(requestExtPrebid *openrtb_ext.ExtRequestPrebid) extCacheInstructions {
	//returnCreative defaults to true
	cacheInstructions := extCacheInstructions{returnCreative: true}
	foundBidsRC := false
	foundVastRC := false

	if requestExtPrebid != nil && requestExtPrebid.Cache != nil {
		if requestExtPrebid.Cache.Bids != nil {
			cacheInstructions.cacheBids = true
			if requestExtPrebid.Cache.Bids.ReturnCreative != nil {
				cacheInstructions.returnCreative = *requestExtPrebid.Cache.Bids.ReturnCreative
				foundBidsRC = true
			}
		}

		if requestExtPrebid.Cache.VastXML != nil {
			cacheInstructions.cacheVAST = true
			if requestExtPrebid.Cache.VastXML.ReturnCreative != nil {
				cacheInstructions.returnCreative = *requestExtPrebid.Cache.VastXML.ReturnCreative
				foundVastRC = true
			}
		}
	}

	if foundBidsRC && foundVastRC {
		cacheInstructions.returnCreative = *requestExtPrebid.Cache.Bids.ReturnCreative || *requestExtPrebid.Cache.VastXML.ReturnCreative
	}

	return cacheInstructions
}

func getExtTargetData(requestExtPrebid *openrtb_ext.ExtRequestPrebid, cacheInstructions extCacheInstructions) *targetData {
	if requestExtPrebid != nil && requestExtPrebid.Targeting != nil {
		return &targetData{
			includeWinners:            *requestExtPrebid.Targeting.IncludeWinners,
			includeBidderKeys:         *requestExtPrebid.Targeting.IncludeBidderKeys,
			includeCacheBids:          cacheInstructions.cacheBids,
			includeCacheVast:          cacheInstructions.cacheVAST,
			includeFormat:             requestExtPrebid.Targeting.IncludeFormat,
			priceGranularity:          *requestExtPrebid.Targeting.PriceGranularity,
			mediaTypePriceGranularity: requestExtPrebid.Targeting.MediaTypePriceGranularity,
			preferDeals:               requestExtPrebid.Targeting.PreferDeals,
		}
	}

	return nil
}

// getDebugInfo returns the boolean flags that allow for debug information in bidResponse.Ext, the SeatBid.httpcalls slice, and
// also sets the debugLog information
func getDebugInfo(test int8, requestExtPrebid *openrtb_ext.ExtRequestPrebid, accountDebugFlag bool, debugLog *DebugLog) (bool, bool, *DebugLog) {
	requestDebugAllow := parseRequestDebugValues(test, requestExtPrebid)
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

func parseRequestDebugValues(test int8, requestExtPrebid *openrtb_ext.ExtRequestPrebid) bool {
	return test == 1 || (requestExtPrebid != nil && requestExtPrebid.Debug)
}

func getExtBidAdjustmentFactors(requestExtPrebid *openrtb_ext.ExtRequestPrebid) map[string]float64 {
	if requestExtPrebid != nil {
		return requestExtPrebid.BidAdjustmentFactors
	}
	return nil
}

func applyFPD(fpd *firstpartydata.ResolvedFirstPartyData, bidReq *openrtb2.BidRequest) {
	if fpd.Site != nil {
		bidReq.Site = fpd.Site
	}
	if fpd.App != nil {
		bidReq.App = fpd.App
	}
	if fpd.User != nil {
		//BuyerUID is a value obtained between fpd extraction and fpd application.
		//BuyerUID needs to be set back to fpd before applying this fpd to final bidder request
		if bidReq.User != nil && len(bidReq.User.BuyerUID) > 0 {
			fpd.User.BuyerUID = bidReq.User.BuyerUID
		}
		bidReq.User = fpd.User
	}
}

func buildBidResponseRequest(req *openrtb2.BidRequest,
	bidderImpResponses stored_responses.BidderImpsWithBidResponses,
	aliases map[string]string,
	bidderImpReplaceImpID stored_responses.BidderImpReplaceImpID) map[openrtb_ext.BidderName]BidderRequest {

	bidderToBidderResponse := make(map[openrtb_ext.BidderName]BidderRequest)

	for bidderName, impResps := range bidderImpResponses {
		resolvedBidder := resolveBidder(string(bidderName), aliases)
		bidderToBidderResponse[bidderName] = BidderRequest{
			BidRequest:            req,
			BidderCoreName:        resolvedBidder,
			BidderName:            bidderName,
			BidderStoredResponses: impResps,
			ImpReplaceImpId:       bidderImpReplaceImpID[string(resolvedBidder)],
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

func setLegacyGDPRFromGPP(r *openrtb2.BidRequest, gpp gpplib.GppContainer) {
	if r.Regs != nil && r.Regs.GDPR == nil {
		if r.Regs.GPPSID != nil {
			// Set to 0 unless SID exists
			regs := *r.Regs
			regs.GDPR = ptrutil.ToPtr[int8](0)
			for _, id := range r.Regs.GPPSID {
				if id == int8(gppConstants.SectionTCFEU2) {
					regs.GDPR = ptrutil.ToPtr[int8](1)
				}
			}
			r.Regs = &regs
		}
	}

	if r.User == nil || len(r.User.Consent) == 0 {
		for _, sec := range gpp.Sections {
			if sec.GetID() == gppConstants.SectionTCFEU2 {
				var user openrtb2.User
				if r.User == nil {
					user = openrtb2.User{}
				} else {
					user = *r.User
				}
				user.Consent = sec.GetValue()
				r.User = &user
			}
		}
	}

}
func setLegacyUSPFromGPP(r *openrtb2.BidRequest, gpp gpplib.GppContainer) {
	if r.Regs == nil {
		return
	}

	if len(r.Regs.USPrivacy) > 0 || r.Regs.GPPSID == nil {
		return
	}
	for _, sid := range r.Regs.GPPSID {
		if sid == int8(gppConstants.SectionUSPV1) {
			for _, sec := range gpp.Sections {
				if sec.GetID() == gppConstants.SectionUSPV1 {
					regs := *r.Regs
					regs.USPrivacy = sec.GetValue()
					r.Regs = &regs
				}
			}
		}
	}

}

func WrapJSONInData(data []byte) []byte {
	res := make([]byte, 0, len(data))
	res = append(res, []byte(`{"data":`)...)
	res = append(res, data...)
	res = append(res, []byte(`}`)...)
	return res
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	mType := bid.MType
	var bidType openrtb_ext.BidType
	if mType > 0 {
		switch mType {
		case openrtb2.MarkupBanner:
			bidType = openrtb_ext.BidTypeBanner
		case openrtb2.MarkupVideo:
			bidType = openrtb_ext.BidTypeVideo
		case openrtb2.MarkupAudio:
			bidType = openrtb_ext.BidTypeAudio
		case openrtb2.MarkupNative:
			bidType = openrtb_ext.BidTypeNative
		default:
			return bidType, fmt.Errorf("Failed to parse bid mType for impression \"%s\"", bid.ImpID)
		}
	} else {
		var err error
		bidType, err = getPrebidMediaTypeForBid(bid)
		if err != nil {
			return bidType, err
		}
	}
	return bidType, nil
}

func getPrebidMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	var err error
	var bidType openrtb_ext.BidType

	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err = json.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			if bidType, err = openrtb_ext.ParseBidType(string(bidExt.Prebid.Type)); err == nil {
				return bidType, nil
			}
		}
	}

	errMsg := fmt.Sprintf("Failed to parse bid mediatype for impression \"%s\"", bid.ImpID)
	if err != nil {
		errMsg = fmt.Sprintf("%s, %s", errMsg, err.Error())
	}

	return bidType, &errortypes.BadServerResponse{
		Message: errMsg,
	}
}
