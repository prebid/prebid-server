package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/prebid/go-gdpr/vendorconsent"
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/firstpartydata"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/privacy/ccpa"
	"github.com/prebid/prebid-server/v3/privacy/lmt"
	"github.com/prebid/prebid-server/v3/schain"
	"github.com/prebid/prebid-server/v3/stored_responses"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

var errInvalidRequestExt = errors.New("request.ext is invalid")

var channelTypeMap = map[metrics.RequestType]config.ChannelType{
	metrics.ReqTypeAMP:       config.ChannelAMP,
	metrics.ReqTypeORTB2App:  config.ChannelApp,
	metrics.ReqTypeVideo:     config.ChannelVideo,
	metrics.ReqTypeORTB2Web:  config.ChannelWeb,
	metrics.ReqTypeORTB2DOOH: config.ChannelDOOH,
}

const unknownBidder string = ""

type requestSplitter struct {
	bidderToSyncerKey map[string]string
	me                metrics.MetricsEngine
	privacyConfig     config.Privacy
	gdprPermsBuilder  gdpr.PermissionsBuilder
	hostSChainNode    *openrtb2.SupplyChainNode
	bidderInfo        config.BidderInfos
	requestValidator  ortb.RequestValidator
}

// cleanOpenRTBRequests splits the input request into requests which are sanitized for each bidder. Intended behavior is:
//
//  1. BidRequest.Imp[].Ext will only contain the "prebid" field and a "bidder" field which has the params for the intended Bidder.
//  2. Every BidRequest.Imp[] requested Bids from the Bidder who keys it.
//  3. BidRequest.User.BuyerUID will be set to that Bidder's ID.
func (rs *requestSplitter) cleanOpenRTBRequests(ctx context.Context,
	auctionReq AuctionRequest,
	requestExt *openrtb_ext.ExtRequest,
	gdprSignal gdpr.Signal,
	gdprEnforced bool,
	bidAdjustmentFactors map[string]float64,
) (bidderRequests []BidderRequest, privacyLabels metrics.PrivacyLabels, errs []error) {
	req := auctionReq.BidRequestWrapper
	if err := PreloadExts(req); err != nil {
		return
	}

	requestAliases, requestAliasesGVLIDs, errs := getRequestAliases(req)
	if len(errs) > 0 {
		return
	}

	bidderImpWithBidResp := stored_responses.InitStoredBidResponses(req.BidRequest, auctionReq.StoredBidResponses)
	hasStoredAuctionResponses := len(auctionReq.StoredAuctionResponses) > 0

	impsByBidder, err := splitImps(req.BidRequest.Imp, rs.requestValidator, requestAliases, hasStoredAuctionResponses, auctionReq.StoredBidResponses)
	if err != nil {
		errs = []error{err}
		return
	}

	explicitBuyerUIDs, err := extractAndCleanBuyerUIDs(req)
	if err != nil {
		errs = []error{err}
		return
	}

	lowerCaseExplicitBuyerUIDs := make(map[string]string)
	for bidder, uid := range explicitBuyerUIDs {
		lowerKey := strings.ToLower(bidder)
		lowerCaseExplicitBuyerUIDs[lowerKey] = uid
	}

	bidderParamsInReqExt, err := ExtractReqExtBidderParamsMap(req.BidRequest)
	if err != nil {
		errs = []error{err}
		return
	}

	sChainWriter, err := schain.NewSChainWriter(requestExt, rs.hostSChainNode)
	if err != nil {
		errs = []error{err}
		return
	}

	var gpp gpplib.GppContainer
	if req.BidRequest.Regs != nil && len(req.BidRequest.Regs.GPP) > 0 {
		var gppErrs []error
		gpp, gppErrs = gpplib.Parse(req.BidRequest.Regs.GPP)
		if len(gppErrs) > 0 {
			errs = append(errs, gppErrs[0])
		}
	}

	consent, err := getConsent(req, gpp)
	if err != nil {
		errs = append(errs, err)
	}

	ccpaEnforcer, err := extractCCPA(req.BidRequest, rs.privacyConfig, &auctionReq.Account, requestAliases, channelTypeMap[auctionReq.LegacyLabels.RType], gpp)
	if err != nil {
		errs = append(errs, err)
	}

	lmtEnforcer := extractLMT(req.BidRequest, rs.privacyConfig)

	// request level privacy policies
	coppa := req.BidRequest.Regs != nil && req.BidRequest.Regs.COPPA == 1
	lmt := lmtEnforcer.ShouldEnforce(unknownBidder)

	privacyLabels.CCPAProvided = ccpaEnforcer.CanEnforce()
	privacyLabels.CCPAEnforced = ccpaEnforcer.ShouldEnforce(unknownBidder)
	privacyLabels.COPPAEnforced = coppa
	privacyLabels.LMTEnforced = lmt

	var gdprPerms gdpr.Permissions = &gdpr.AlwaysAllow{}

	if gdprEnforced {
		privacyLabels.GDPREnforced = true
		parsedConsent, err := vendorconsent.ParseString(consent)
		if err == nil {
			version := int(parsedConsent.Version())
			privacyLabels.GDPRTCFVersion = metrics.TCFVersionToValue(version)
		}

		gdprRequestInfo := gdpr.RequestInfo{
			AliasGVLIDs: requestAliasesGVLIDs,
			Consent:     consent,
			GDPRSignal:  gdprSignal,
			PublisherID: auctionReq.LegacyLabels.PubID,
		}
		gdprPerms = rs.gdprPermsBuilder(auctionReq.TCF2Config, gdprRequestInfo)
	}

	bidderRequests = make([]BidderRequest, 0, len(impsByBidder))

	for bidder, imps := range impsByBidder {
		fpdUserEIDsPresent := fpdUserEIDExists(req, auctionReq.FirstPartyData, bidder)
		reqWrapperCopy := req.CloneAndClearImpWrappers()
		bidRequestCopy := *req.BidRequest
		reqWrapperCopy.BidRequest = &bidRequestCopy
		reqWrapperCopy.Imp = imps

		coreBidder, isRequestAlias := resolveBidder(bidder, requestAliases)

		// apply bidder-specific schains
		sChainWriter.Write(reqWrapperCopy, bidder)

		// eid scrubbing
		if err := removeUnpermissionedEids(reqWrapperCopy, bidder); err != nil {
			errs = append(errs, fmt.Errorf("unable to enforce request.ext.prebid.data.eidpermissions because %v", err))
			continue
		}

		// generate bidder-specific request ext
		err = buildRequestExtForBidder(bidder, reqWrapperCopy, bidderParamsInReqExt, auctionReq.Account.AlternateBidderCodes)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// apply bid adjustments
		if auctionReq.Account.PriceFloors.IsAdjustForBidAdjustmentEnabled() {
			applyBidAdjustmentToFloor(reqWrapperCopy, bidder, bidAdjustmentFactors)
		}

		// prepare user
		syncerKey := rs.bidderToSyncerKey[string(coreBidder)]
		hadSync := prepareUser(reqWrapperCopy, bidder, syncerKey, lowerCaseExplicitBuyerUIDs, auctionReq.UserSyncs)

		auctionPermissions := gdprPerms.AuctionActivitiesAllowed(ctx, coreBidder, openrtb_ext.BidderName(bidder))

		// privacy blocking
		if rs.isBidderBlockedByPrivacy(reqWrapperCopy, auctionReq.Activities, auctionPermissions, coreBidder, openrtb_ext.BidderName(bidder)) {
			continue
		}

		// fpd
		applyFPD(auctionReq.FirstPartyData, coreBidder, openrtb_ext.BidderName(bidder), isRequestAlias, reqWrapperCopy, fpdUserEIDsPresent)

		// privacy scrubbing
		if err := rs.applyPrivacy(reqWrapperCopy, coreBidder, bidder, auctionReq, auctionPermissions, ccpaEnforcer, lmt, coppa); err != nil {
			errs = append(errs, err)
			continue
		}

		// GPP downgrade: always downgrade unless we can confirm GPP is supported
		if shouldSetLegacyPrivacy(rs.bidderInfo, string(coreBidder)) {
			setLegacyGDPRFromGPP(reqWrapperCopy, gpp)
			setLegacyUSPFromGPP(reqWrapperCopy, gpp)
		}

		// remove imps with stored responses so they aren't sent to the bidder
		if impResponses, ok := bidderImpWithBidResp[openrtb_ext.BidderName(bidder)]; ok {
			removeImpsWithStoredResponses(reqWrapperCopy, impResponses)
		}

		// down convert
		info, ok := rs.bidderInfo[bidder]
		if !ok || info.OpenRTB == nil || info.OpenRTB.Version != "2.6" {
			reqWrapperCopy.Regs = ortb.CloneRegs(reqWrapperCopy.Regs)
			if err := openrtb_ext.ConvertDownTo25(reqWrapperCopy); err != nil {
				errs = append(errs, err)
				continue
			}
		}

		// sync wrapper
		if err := reqWrapperCopy.RebuildRequest(); err != nil {
			errs = append(errs, err)
			continue
		}

		// choose labels
		bidderLabels := metrics.AdapterLabels{
			Adapter: coreBidder,
		}
		if !hadSync && req.BidRequest.App == nil {
			bidderLabels.CookieFlag = metrics.CookieFlagNo
		} else {
			bidderLabels.CookieFlag = metrics.CookieFlagYes
		}
		if len(reqWrapperCopy.Imp) > 0 {
			bidderLabels.Source = auctionReq.LegacyLabels.Source
			bidderLabels.RType = auctionReq.LegacyLabels.RType
			bidderLabels.PubID = auctionReq.LegacyLabels.PubID
			bidderLabels.CookieFlag = auctionReq.LegacyLabels.CookieFlag
			bidderLabels.AdapterBids = metrics.AdapterBidPresent
		}

		bidderRequest := BidderRequest{
			BidderName:            openrtb_ext.BidderName(bidder),
			BidderCoreName:        coreBidder,
			BidRequest:            reqWrapperCopy.BidRequest,
			IsRequestAlias:        isRequestAlias,
			BidderStoredResponses: bidderImpWithBidResp[openrtb_ext.BidderName(bidder)],
			ImpReplaceImpId:       auctionReq.BidderImpReplaceImpID[bidder],
			BidderLabels:          bidderLabels,
		}
		bidderRequests = append(bidderRequests, bidderRequest)
	}

	return
}

// fpdUserEIDExists determines if req fpd config had User.EIDs
func fpdUserEIDExists(req *openrtb_ext.RequestWrapper, fpd map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData, bidder string) bool {
	fpdToApply, exists := fpd[openrtb_ext.BidderName(bidder)]
	if !exists || fpdToApply == nil {
		return false
	}
	if fpdToApply.User == nil {
		return false
	}
	fpdUserEIDs := fpdToApply.User.EIDs

	if len(fpdUserEIDs) == 0 {
		return false
	}
	if req.User == nil {
		return true
	}

	reqUserEIDs := req.User.EIDs

	if len(reqUserEIDs) != len(fpdUserEIDs) {
		return true
	}

	// if bidder fpd didn't have user.eids then user.eids will remain the same
	// hence we can use the same index to compare elements
	for i := range reqUserEIDs {
		pReqUserEID := &reqUserEIDs[i]
		pFpdUserEID := &fpdUserEIDs[i]
		if pReqUserEID != pFpdUserEID {
			return true
		}
	}
	return false
}

// removeImpsWithStoredResponses deletes imps with stored bid resp
func removeImpsWithStoredResponses(req *openrtb_ext.RequestWrapper, impBidResponses map[string]json.RawMessage) {
	if len(impBidResponses) == 0 {
		return
	}

	imps := req.Imp
	req.Imp = nil //to indicate this bidder doesn't have real requests
	for _, imp := range imps {
		if _, ok := impBidResponses[imp.ID]; !ok {
			//add real imp back to request
			req.Imp = append(req.Imp, imp)
		}
	}
}

// PreloadExts ensures all exts have been unmarshalled into wrapper ext objects
func PreloadExts(req *openrtb_ext.RequestWrapper) error {
	if req == nil {
		return nil
	}
	if _, err := req.GetRequestExt(); err != nil {
		return err
	}
	if _, err := req.GetUserExt(); err != nil {
		return err
	}
	if _, err := req.GetDeviceExt(); err != nil {
		return err
	}
	if _, err := req.GetRegExt(); err != nil {
		return err
	}
	if _, err := req.GetSiteExt(); err != nil {
		return err
	}
	if _, err := req.GetDOOHExt(); err != nil {
		return err
	}
	if _, err := req.GetSourceExt(); err != nil {
		return err
	}
	return nil
}

func (rs *requestSplitter) isBidderBlockedByPrivacy(r *openrtb_ext.RequestWrapper, activities privacy.ActivityControl, auctionPermissions gdpr.AuctionPermissions, coreBidder, bidderName openrtb_ext.BidderName) bool {
	// activities control
	scope := privacy.Component{Type: privacy.ComponentTypeBidder, Name: bidderName.String()}
	fetchBidsActivityAllowed := activities.Allow(privacy.ActivityFetchBids, scope, privacy.NewRequestFromBidRequest(*r))
	if !fetchBidsActivityAllowed {
		return true
	}

	// gdpr
	if !auctionPermissions.AllowBidRequest {
		rs.me.RecordAdapterGDPRRequestBlocked(coreBidder)
		return true
	}

	return false
}

func (rs *requestSplitter) applyPrivacy(reqWrapper *openrtb_ext.RequestWrapper, coreBidderName openrtb_ext.BidderName, bidderName string, auctionReq AuctionRequest, auctionPermissions gdpr.AuctionPermissions, ccpaEnforcer privacy.PolicyEnforcer, lmt bool, coppa bool) error {
	scope := privacy.Component{Type: privacy.ComponentTypeBidder, Name: bidderName}
	ipConf := privacy.IPConf{IPV6: auctionReq.Account.Privacy.IPv6Config, IPV4: auctionReq.Account.Privacy.IPv4Config}

	bidRequest := ortb.CloneBidRequestPartial(reqWrapper.BidRequest)
	reqWrapper.BidRequest = bidRequest

	passIDActivityAllowed := auctionReq.Activities.Allow(privacy.ActivityTransmitUserFPD, scope, privacy.NewRequestFromBidRequest(*reqWrapper))
	buyerUIDSet := reqWrapper.User != nil && reqWrapper.User.BuyerUID != ""
	buyerUIDRemoved := false
	if !passIDActivityAllowed {
		privacy.ScrubUserFPD(reqWrapper)
		buyerUIDRemoved = true
	} else {
		if !auctionPermissions.PassID {
			privacy.ScrubGdprID(reqWrapper)
			buyerUIDRemoved = true
		}

		if ccpaEnforcer.ShouldEnforce(bidderName) {
			privacy.ScrubDeviceIDsIPsUserDemoExt(reqWrapper, ipConf, "eids", false)
			buyerUIDRemoved = true
		}
	}
	if buyerUIDSet && buyerUIDRemoved {
		rs.me.RecordAdapterBuyerUIDScrubbed(coreBidderName)
	}

	passGeoActivityAllowed := auctionReq.Activities.Allow(privacy.ActivityTransmitPreciseGeo, scope, privacy.NewRequestFromBidRequest(*reqWrapper))
	if !passGeoActivityAllowed {
		privacy.ScrubGeoAndDeviceIP(reqWrapper, ipConf)
	} else {
		if !auctionPermissions.PassGeo {
			privacy.ScrubGeoAndDeviceIP(reqWrapper, ipConf)
		}
		if ccpaEnforcer.ShouldEnforce(bidderName) {
			privacy.ScrubDeviceIDsIPsUserDemoExt(reqWrapper, ipConf, "eids", false)
		}
	}

	if lmt || coppa {
		privacy.ScrubDeviceIDsIPsUserDemoExt(reqWrapper, ipConf, "eids", coppa)
	}

	passTIDAllowed := auctionReq.Activities.Allow(privacy.ActivityTransmitTIDs, scope, privacy.NewRequestFromBidRequest(*reqWrapper))
	if !passTIDAllowed {
		privacy.ScrubTID(reqWrapper)
	}

	if err := reqWrapper.RebuildRequest(); err != nil {
		return err
	}

	// *bidRequest = *reqWrapper.BidRequest
	return nil
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

func extractCCPA(orig *openrtb2.BidRequest, privacyConfig config.Privacy, account *config.Account, requestAliases map[string]string, requestType config.ChannelType, gpp gpplib.GppContainer) (privacy.PolicyEnforcer, error) {
	// Quick extra wrapper until RequestWrapper makes its way into CleanRequests
	ccpaPolicy, err := ccpa.ReadFromRequestWrapper(&openrtb_ext.RequestWrapper{BidRequest: orig}, gpp)
	if err != nil {
		return privacy.NilPolicyEnforcer{}, err
	}

	validBidders := GetValidBidders(requestAliases)
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
		err := jsonutil.Unmarshal(bidRequest.Ext, &reqExt)
		if err != nil {
			return nil, fmt.Errorf("error decoding Request.ext : %s", err.Error())
		}
	}

	if reqExt.Prebid.BidderParams == nil {
		return nil, nil
	}

	var bidderParams map[string]json.RawMessage
	err := jsonutil.Unmarshal(reqExt.Prebid.BidderParams, &bidderParams)
	if err != nil {
		return nil, err
	}

	return bidderParams, nil
}

func buildRequestExtForBidder(bidder string, req *openrtb_ext.RequestWrapper, reqExtBidderParams map[string]json.RawMessage, cfgABC *openrtb_ext.ExtAlternateBidderCodes) error {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return err
	}
	prebid := reqExt.GetPrebid()

	// Resolve Alternate Bidder Codes
	var reqABC *openrtb_ext.ExtAlternateBidderCodes
	if prebid != nil && prebid.AlternateBidderCodes != nil {
		reqABC = prebid.AlternateBidderCodes
	}
	alternateBidderCodes := buildRequestExtAlternateBidderCodes(bidder, cfgABC, reqABC)

	// Build New/Filtered Prebid Ext
	prebidNew := openrtb_ext.ExtRequestPrebid{
		BidderParams:         reqExtBidderParams[bidder],
		AlternateBidderCodes: alternateBidderCodes,
	}

	// Copy Allowed Fields
	// Per: https://docs.prebid.org/prebid-server/endpoints/openrtb2/pbs-endpoint-auction.html#prebid-server-ortb2-extension-summary
	if prebid != nil {
		prebidNew.Channel = prebid.Channel
		prebidNew.CurrencyConversions = prebid.CurrencyConversions
		prebidNew.Debug = prebid.Debug
		prebidNew.Integration = prebid.Integration
		prebidNew.MultiBid = buildRequestExtMultiBid(bidder, prebid.MultiBid, alternateBidderCodes)
		prebidNew.Sdk = prebid.Sdk
		prebidNew.Server = prebid.Server
		prebidNew.Targeting = buildRequestExtTargeting(prebid.Targeting)
	}

	reqExt.SetPrebid(&prebidNew)
	return nil
}

func buildRequestExtAlternateBidderCodes(bidder string, accABC *openrtb_ext.ExtAlternateBidderCodes, reqABC *openrtb_ext.ExtAlternateBidderCodes) *openrtb_ext.ExtAlternateBidderCodes {
	if altBidderCodes := copyExtAlternateBidderCodes(bidder, reqABC); altBidderCodes != nil {
		return altBidderCodes
	}

	if altBidderCodes := copyExtAlternateBidderCodes(bidder, accABC); altBidderCodes != nil {
		return altBidderCodes
	}

	return nil
}

func copyExtAlternateBidderCodes(bidder string, altBidderCodes *openrtb_ext.ExtAlternateBidderCodes) *openrtb_ext.ExtAlternateBidderCodes {
	if altBidderCodes != nil {
		alternateBidderCodes := &openrtb_ext.ExtAlternateBidderCodes{
			Enabled: altBidderCodes.Enabled,
		}

		if bidderCodes, ok := altBidderCodes.IsBidderInAlternateBidderCodes(bidder); ok {
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
			if strings.ToLower(multiBid.Bidder) == adapter || isBidderInExtAlternateBidderCodes(adapter, strings.ToLower(multiBid.Bidder), adapterABC) {
				adapterMultiBid = append(adapterMultiBid, multiBid)
			}
		} else {
			for _, bidder := range multiBid.Bidders {
				if strings.ToLower(bidder) == adapter || isBidderInExtAlternateBidderCodes(adapter, strings.ToLower(bidder), adapterABC) {
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

func buildRequestExtTargeting(t *openrtb_ext.ExtRequestTargeting) *openrtb_ext.ExtRequestTargeting {
	if t == nil || t.IncludeBrandCategory == nil {
		return nil
	}

	// only include fields bidders can use to influence their response and which does
	// not expose information about other bidders or restricted auction processing
	return &openrtb_ext.ExtRequestTargeting{
		IncludeBrandCategory: t.IncludeBrandCategory,
	}
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

// extractAndCleanBuyerUIDs parses the values from user.ext.prebid.buyeruids, and then deletes those values from the ext.
// This prevents a Bidder from using these values to figure out who else is involved in the Auction.
func extractAndCleanBuyerUIDs(req *openrtb_ext.RequestWrapper) (map[string]string, error) {
	if req.User == nil {
		return nil, nil
	}

	userExt, err := req.GetUserExt()
	if err != nil {
		return nil, err
	}

	prebid := userExt.GetPrebid()
	if prebid == nil {
		return nil, nil
	}

	buyerUIDs := prebid.BuyerUIDs

	prebid.BuyerUIDs = nil
	userExt.SetPrebid(prebid)

	// The API guarantees that user.ext.prebid.buyeruids exists and has at least one ID defined,
	// as long as user.ext.prebid exists.
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
func splitImps(imps []openrtb2.Imp, requestValidator ortb.RequestValidator, requestAliases map[string]string, hasStoredAuctionResponses bool, storedBidResponses stored_responses.ImpBidderStoredResp) (map[string][]openrtb2.Imp, error) {
	bidderImps := make(map[string][]openrtb2.Imp)

	for i, imp := range imps {
		var impExt map[string]json.RawMessage
		if err := jsonutil.UnmarshalValid(imp.Ext, &impExt); err != nil {
			return nil, fmt.Errorf("invalid json for imp[%d]: %v", i, err)
		}

		var impExtPrebid map[string]json.RawMessage
		if impExtPrebidJSON, exists := impExt[openrtb_ext.PrebidExtKey]; exists {
			// validation already performed by impExt unmarshal. no error is possible here, proven by tests.
			jsonutil.Unmarshal(impExtPrebidJSON, &impExtPrebid)
		}

		var impExtPrebidBidder map[string]json.RawMessage
		if impExtPrebidBidderJSON, exists := impExtPrebid[openrtb_ext.PrebidExtBidderKey]; exists {
			// validation already performed by impExt unmarshal. no error is possible here, proven by tests.
			jsonutil.Unmarshal(impExtPrebidBidderJSON, &impExtPrebidBidder)
		}

		var impExtPrebidImp map[string]json.RawMessage
		if impExtPrebidImpJSON, exists := impExtPrebid["imp"]; exists {
			jsonutil.Unmarshal(impExtPrebidImpJSON, &impExtPrebidImp)
		}

		sanitizedImpExt, err := createSanitizedImpExt(impExt, impExtPrebid)
		if err != nil {
			return nil, fmt.Errorf("unable to remove other bidder fields for imp[%d]: %v", i, err)
		}

		for bidder, bidderExt := range impExtPrebidBidder {
			impCopy := imp

			if impBidderFPD, exists := impExtPrebidImp[bidder]; exists {
				if err := mergeImpFPD(&impCopy, impBidderFPD, i); err != nil {
					return nil, err
				}
				impWrapper := openrtb_ext.ImpWrapper{Imp: &impCopy}
				cfg := ortb.ValidationConfig{
					SkipBidderParams: true,
					SkipNative:       true,
				}
				if err := requestValidator.ValidateImp(&impWrapper, cfg, i, requestAliases, hasStoredAuctionResponses, storedBidResponses); err != nil {
					return nil, &errortypes.InvalidImpFirstPartyData{
						Message: fmt.Sprintf("merging bidder imp first party data for imp %s results in an invalid imp: %v", imp.ID, err),
					}
				}
			}

			sanitizedImpExt[openrtb_ext.PrebidExtBidderKey] = bidderExt

			impExtJSON, err := jsonutil.Marshal(sanitizedImpExt)
			if err != nil {
				return nil, fmt.Errorf("unable to remove other bidder fields for imp[%d]: cannot marshal ext: %v", i, err)
			}
			impCopy.Ext = impExtJSON

			bidderImps[bidder] = append(bidderImps[bidder], impCopy)
		}
	}

	return bidderImps, nil
}

func mergeImpFPD(imp *openrtb2.Imp, fpd json.RawMessage, index int) error {
	if err := jsonutil.MergeClone(imp, fpd); err != nil {
		if strings.Contains(err.Error(), "invalid json on existing object") {
			return fmt.Errorf("invalid imp ext for imp[%d]", index)
		}
		return fmt.Errorf("invalid first party data for imp[%d]", index)
	}
	return nil
}

var allowedImpExtPrebidFields = map[string]interface{}{
	openrtb_ext.IsRewardedInventoryKey: struct{}{},
	openrtb_ext.OptionsKey:             struct{}{},
}

var deniedImpExtFields = map[string]interface{}{
	openrtb_ext.PrebidExtKey: struct{}{},
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
		if impExtPrebidJSON, err := jsonutil.Marshal(sanitizedImpPrebidExt); err == nil {
			sanitizedImpExt[openrtb_ext.PrebidExtKey] = impExtPrebidJSON
		} else {
			return nil, fmt.Errorf("cannot marshal ext.prebid: %v", err)
		}
	}

	// copy reserved imp[].ext fields known to not be bidder names
	for k, v := range impExt {
		if _, exists := deniedImpExtFields[k]; !exists {
			sanitizedImpExt[k] = v
		}
	}

	return sanitizedImpExt, nil
}

// prepareUser changes req.User so that it's ready for the given bidder.
// In this function, "givenBidder" may or may not be an alias. "coreBidder" must *not* be an alias.
// It returns true if a Cookie User Sync existed, and false otherwise.
func prepareUser(req *openrtb_ext.RequestWrapper, givenBidder, syncerKey string, explicitBuyerUIDs map[string]string, usersyncs IdFetcher) bool {
	cookieId, hadCookie, _ := usersyncs.GetUID(syncerKey)

	if id, ok := explicitBuyerUIDs[strings.ToLower(givenBidder)]; ok {
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

// removeUnpermissionedEids modifies the request to remove any request.user.eids not permissions for the specific bidder
func removeUnpermissionedEids(reqWrapper *openrtb_ext.RequestWrapper, bidder string) error {
	// ensure request might have eids (as much as we can check before unmarshalling)
	if reqWrapper.User == nil || len(reqWrapper.User.EIDs) == 0 {
		return nil
	}

	// ensure request has eid permissions to enforce
	reqExt, err := reqWrapper.GetRequestExt()
	if err != nil {
		return err
	}
	if reqExt == nil {
		return nil
	}

	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid == nil || reqExtPrebid.Data == nil || len(reqExtPrebid.Data.EidPermissions) == 0 {
		return nil
	}

	eids := reqWrapper.User.EIDs

	// translate eid permissions to a map for quick lookup
	eidRules := make(map[string][]string)
	for _, p := range reqExtPrebid.Data.EidPermissions {
		eidRules[p.Source] = p.Bidders
	}

	eidsAllowed := make([]openrtb2.EID, 0, len(eids))
	for _, eid := range eids {
		allowed := false
		if rule, hasRule := eidRules[eid.Source]; hasRule {
			for _, ruleBidder := range rule {
				if ruleBidder == "*" || strings.EqualFold(ruleBidder, bidder) {
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

	if len(eidsAllowed) == 0 {
		reqWrapper.User.EIDs = nil
	} else {
		reqWrapper.User.EIDs = eidsAllowed
	}
	return nil
}

// resolveBidder returns the known BidderName associated with bidder, if bidder is an alias. If it's not an alias, the bidder is returned.
func resolveBidder(bidder string, requestAliases map[string]string) (openrtb_ext.BidderName, bool) {
	normalisedBidderName, _ := openrtb_ext.NormalizeBidderName(bidder)

	if coreBidder, ok := requestAliases[bidder]; ok {
		return openrtb_ext.BidderName(coreBidder), true
	}

	return normalisedBidderName, false
}

func getRequestAliases(req *openrtb_ext.RequestWrapper) (map[string]string, map[string]uint16, []error) {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return nil, nil, []error{errInvalidRequestExt}
	}

	if prebid := reqExt.GetPrebid(); prebid != nil {
		return prebid.Aliases, prebid.AliasGVLIDs, nil
	}

	return nil, nil, nil
}

func GetValidBidders(requestAliases map[string]string) map[string]struct{} {
	validBidders := openrtb_ext.BuildBidderNameHashSet()

	for k := range requestAliases {
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

func getExtTargetData(requestExtPrebid *openrtb_ext.ExtRequestPrebid, cacheInstructions extCacheInstructions, account config.Account) (*targetData, []*errortypes.Warning) {
	if requestExtPrebid != nil && requestExtPrebid.Targeting != nil {
		prefix, warning := getTargetDataPrefix(requestExtPrebid.Targeting.Prefix, account)
		return &targetData{
			alwaysIncludeDeals:        requestExtPrebid.Targeting.AlwaysIncludeDeals,
			includeBidderKeys:         ptrutil.ValueOrDefault(requestExtPrebid.Targeting.IncludeBidderKeys),
			includeCacheBids:          cacheInstructions.cacheBids,
			includeCacheVast:          cacheInstructions.cacheVAST,
			includeFormat:             requestExtPrebid.Targeting.IncludeFormat,
			includeWinners:            ptrutil.ValueOrDefault(requestExtPrebid.Targeting.IncludeWinners),
			mediaTypePriceGranularity: ptrutil.ValueOrDefault(requestExtPrebid.Targeting.MediaTypePriceGranularity),
			preferDeals:               requestExtPrebid.Targeting.PreferDeals,
			priceGranularity:          ptrutil.ValueOrDefault(requestExtPrebid.Targeting.PriceGranularity),
			prefix:                    prefix,
		}, warning
	}

	return nil, nil
}

func getTargetDataPrefix(requestPrefix string, account config.Account) (string, []*errortypes.Warning) {
	var warnings []*errortypes.Warning

	maxLength := MaxKeyLength
	if account.TruncateTargetAttribute != nil {
		if *account.TruncateTargetAttribute > MinKeyLength {
			maxLength = *account.TruncateTargetAttribute
		}

		if *account.TruncateTargetAttribute < MinKeyLength {
			warnings = append(warnings, &errortypes.Warning{
				WarningCode: errortypes.TooShortTargetingPrefixWarningCode,
				Message:     "targeting prefix is shorter than 'MinKeyLength' value: increase prefix length",
			})
			return DefaultKeyPrefix, warnings
		}
	}

	maxLength -= MinKeyLength
	result := DefaultKeyPrefix

	if requestPrefix != "" {
		result = requestPrefix
	} else if account.TargetingPrefix != "" {
		result = account.TargetingPrefix
	}

	if len(result) > maxLength {
		warnings = append(warnings, &errortypes.Warning{
			WarningCode: errortypes.TooLongTargetingPrefixWarningCode,
			Message:     "targeting prefix combined with key attribute is longer than 'settings.targeting.truncate-attr-chars' value: decrease prefix length or increase truncate-attr-chars",
		})
		return DefaultKeyPrefix, warnings
	}

	return result, warnings
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
	if requestExtPrebid != nil && requestExtPrebid.BidAdjustmentFactors != nil {
		caseInsensitiveMap := make(map[string]float64, len(requestExtPrebid.BidAdjustmentFactors))
		for bidder, bidAdjFactor := range requestExtPrebid.BidAdjustmentFactors {
			caseInsensitiveMap[strings.ToLower(bidder)] = bidAdjFactor
		}
		return caseInsensitiveMap
	}
	return nil
}

func applyFPD(fpd map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData,
	coreBidderName openrtb_ext.BidderName,
	bidderName openrtb_ext.BidderName,
	isRequestAlias bool,
	reqWrapper *openrtb_ext.RequestWrapper,
	fpdUserEIDsPresent bool) {
	if fpd == nil {
		return
	}

	bidder := coreBidderName
	if isRequestAlias {
		bidder = bidderName
	}

	fpdToApply, exists := fpd[bidder]
	if !exists || fpdToApply == nil {
		return
	}

	if fpdToApply.Site != nil {
		reqWrapper.Site = fpdToApply.Site
	}

	if fpdToApply.App != nil {
		reqWrapper.App = fpdToApply.App
	}

	if fpdToApply.Device != nil {
		reqWrapper.Device = fpdToApply.Device
	}

	if fpdToApply.User != nil {
		if reqWrapper.User != nil {
			if len(reqWrapper.User.BuyerUID) > 0 {
				//BuyerUID is a value obtained between fpd extraction and fpd application.
				//BuyerUID needs to be set back to fpd before applying this fpd to final bidder request
				fpdToApply.User.BuyerUID = reqWrapper.User.BuyerUID
			}

			// if FPD config didn't have user.eids - use reqWrapper.User.EIDs after removeUnpermissionedEids
			if !fpdUserEIDsPresent {
				fpdToApply.User.EIDs = reqWrapper.User.EIDs
			}
		}
		reqWrapper.User = fpdToApply.User
	}
}

func setLegacyGDPRFromGPP(r *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) {
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

func setLegacyUSPFromGPP(r *openrtb_ext.RequestWrapper, gpp gpplib.GppContainer) {
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
		err = jsonutil.Unmarshal(bid.Ext, &bidExt)
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

func applyBidAdjustmentToFloor(req *openrtb_ext.RequestWrapper, bidder string, adjustmentFactors map[string]float64) {
	if len(adjustmentFactors) == 0 {
		return
	}

	bidAdjustment := 1.0
	if v, ok := adjustmentFactors[bidder]; ok && v != 0.0 {
		bidAdjustment = v
	}

	if bidAdjustment != 1.0 {
		for index, imp := range req.Imp {
			imp.BidFloor = imp.BidFloor / bidAdjustment
			req.Imp[index] = imp
		}
	}
}
