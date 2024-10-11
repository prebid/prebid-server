package exchange

import (
	"bytes"
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

	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/firstpartydata"
	"github.com/prebid/prebid-server/v2/gdpr"
	"github.com/prebid/prebid-server/v2/metrics"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/ortb"
	"github.com/prebid/prebid-server/v2/privacy"
	"github.com/prebid/prebid-server/v2/privacy/ccpa"
	"github.com/prebid/prebid-server/v2/privacy/lmt"
	"github.com/prebid/prebid-server/v2/schain"
	"github.com/prebid/prebid-server/v2/stored_responses"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
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

	explicitBuyerUIDs, err := extractBuyerUIDs(req.BidRequest.User)
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
		reqWrapperCopy := req.Clone() //TODO: check if it is cloning stuff that we don't need to clone
		reqCopy := *req.BidRequest
		reqCopy.Imp = imps

		coreBidder, isRequestAlias := resolveBidder(bidder, requestAliases)

		// apply bidder-specific schains
		sChainWriter.Write(&reqCopy, bidder)

		// generate bidder-specific request ext
		reqCopy.Ext, err = buildRequestExtForBidder(bidder, req.BidRequest.Ext, requestExt, bidderParamsInReqExt, auctionReq.Account.AlternateBidderCodes)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// eid scrubbing
		if err := removeUnpermissionedEids(&reqCopy, bidder, requestExt); err != nil {
			errs = append(errs, fmt.Errorf("unable to enforce request.ext.prebid.data.eidpermissions because %v", err))
			continue
		}

		// apply bid adjustments
		if auctionReq.Account.PriceFloors.IsAdjustForBidAdjustmentEnabled() {
			applyBidAdjustmentToFloor(&reqCopy, bidder, bidAdjustmentFactors)
		}

		// prepare user
		syncerKey := rs.bidderToSyncerKey[string(coreBidder)]
		hadSync := prepareUser(&reqCopy, bidder, syncerKey, lowerCaseExplicitBuyerUIDs, auctionReq.UserSyncs)

		auctionPermissions := gdprPerms.AuctionActivitiesAllowed(ctx, coreBidder, openrtb_ext.BidderName(bidder))

		// privacy blocking
		if rs.isBidderBlockedByPrivacy(req, auctionReq.Activities, auctionPermissions, coreBidder, openrtb_ext.BidderName(bidder)) {
			continue
		}

		// fpd
		applyFPD(auctionReq.FirstPartyData, coreBidder, openrtb_ext.BidderName(bidder), isRequestAlias, &reqCopy)

		// privacy scrubbing
		if err := rs.applyPrivacy(&reqCopy, coreBidder, bidder, auctionReq, auctionPermissions, ccpaEnforcer, lmt, coppa); err != nil {
			errs = append(errs, err)
			continue
		}

		// GPP downgrade: always downgrade unless we can confirm GPP is supported
		if shouldSetLegacyPrivacy(rs.bidderInfo, string(coreBidder)) {
			setLegacyGDPRFromGPP(&reqCopy, gpp)
			setLegacyUSPFromGPP(&reqCopy, gpp)
		}

		//TODO: could make bidderImpWithBidResp[openrtb_ext.BidderName(bidder)]
		if impResponses, ok := bidderImpWithBidResp[openrtb_ext.BidderName(bidder)]; ok {
			removeImpsWithStoredResponses(&reqCopy, impResponses)
		}

		// down convert
		info, ok := rs.bidderInfo[bidder]
		if !ok || info.OpenRTB == nil || info.OpenRTB.Version != "2.6" {
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
		if len(reqCopy.Imp) > 0 {
			bidderLabels.Source = auctionReq.LegacyLabels.Source
			bidderLabels.RType = auctionReq.LegacyLabels.RType
			bidderLabels.PubID = auctionReq.LegacyLabels.PubID
			bidderLabels.CookieFlag = auctionReq.LegacyLabels.CookieFlag
			bidderLabels.AdapterBids = metrics.AdapterBidPresent
		}

		bidderRequest := BidderRequest{
			BidderName:            openrtb_ext.BidderName(bidder),
			BidderCoreName:        coreBidder,
			BidRequest:            &reqCopy,
			IsRequestAlias:        isRequestAlias,
			BidderStoredResponses: bidderImpWithBidResp[openrtb_ext.BidderName(bidder)],
			ImpReplaceImpId:       auctionReq.BidderImpReplaceImpID[bidder],
			BidderLabels:          bidderLabels,
		}
		bidderRequests = append(bidderRequests, bidderRequest)
	}

	return
}

// removeImpsWithStoredResponses deletes imps with stored bid resp
func removeImpsWithStoredResponses(req *openrtb2.BidRequest, impBidResponses map[string]json.RawMessage) {
	imps := req.Imp
	req.Imp = nil //to indicate this bidder doesn't have real requests
	for _, imp := range imps {
		if _, ok := impBidResponses[imp.ID]; !ok {
			//add real imp back to request
			req.Imp = append(req.Imp, imp)
		}
	}
}

// PreloadExts...
//TODO: move elsewhere, perhaps into openrtb_ext package?
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

func (rs *requestSplitter) applyPrivacy(bidRequest *openrtb2.BidRequest, coreBidderName openrtb_ext.BidderName, bidderName string, auctionReq AuctionRequest, auctionPermissions gdpr.AuctionPermissions, ccpaEnforcer privacy.PolicyEnforcer, lmt bool, coppa bool) error {
	scope := privacy.Component{Type: privacy.ComponentTypeBidder, Name: bidderName}
	ipConf := privacy.IPConf{IPV6: auctionReq.Account.Privacy.IPv6Config, IPV4: auctionReq.Account.Privacy.IPv4Config}

	reqWrapper := &openrtb_ext.RequestWrapper{
		BidRequest: ortb.CloneBidRequestPartial(bidRequest),
	}

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

	bidRequest = reqWrapper.BidRequest
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
		prebid.Channel = requestExtParsed.Prebid.Channel
		prebid.CurrencyConversions = requestExtParsed.Prebid.CurrencyConversions
		prebid.Debug = requestExtParsed.Prebid.Debug
		prebid.Integration = requestExtParsed.Prebid.Integration
		prebid.MultiBid = buildRequestExtMultiBid(bidder, requestExtParsed.Prebid.MultiBid, alternateBidderCodes)
		prebid.Sdk = requestExtParsed.Prebid.Sdk
		prebid.Server = requestExtParsed.Prebid.Server
	}

	// Marshal New Prebid Object
	prebidJson, err := jsonutil.Marshal(prebid)
	if err != nil {
		return nil, err
	}

	// Parse Existing Ext
	extMap := make(map[string]json.RawMessage)
	if len(requestExt) != 0 {
		if err := jsonutil.Unmarshal(requestExt, &extMap); err != nil {
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
		return jsonutil.Marshal(extMap)
	} else {
		return nil, nil
	}
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
	if err := jsonutil.Unmarshal(user.Ext, &userExt); err != nil {
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
		if newUserExtBytes, err := jsonutil.Marshal(userExt); err != nil {
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
		if impExtPrebidJSON, err := jsonutil.Marshal(sanitizedImpPrebidExt); err == nil {
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
	if err := jsonutil.Unmarshal(request.User.Ext, &userExt); err != nil {
		return err
	}

	eidsJSON, eidsSpecified := userExt["eids"]
	if !eidsSpecified {
		return nil
	}

	var eids []openrtb2.EID
	if err := jsonutil.Unmarshal(eidsJSON, &eids); err != nil {
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

	// marshal eidsAllowed back to userExt
	if len(eidsAllowed) == 0 {
		delete(userExt, "eids")
	} else {
		eidsRaw, err := jsonutil.Marshal(eidsAllowed)
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

	userExtJSON, err := jsonutil.Marshal(userExt)
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
			alwaysIncludeDeals:        requestExtPrebid.Targeting.AlwaysIncludeDeals,
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
	if requestExtPrebid != nil && requestExtPrebid.BidAdjustmentFactors != nil {
		caseInsensitiveMap := make(map[string]float64, len(requestExtPrebid.BidAdjustmentFactors))
		for bidder, bidAdjFactor := range requestExtPrebid.BidAdjustmentFactors {
			caseInsensitiveMap[strings.ToLower(bidder)] = bidAdjFactor
		}
		return caseInsensitiveMap
	}
	return nil
}

func applyFPD(fpd map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData, coreBidderName openrtb_ext.BidderName, bidderName openrtb_ext.BidderName, isRequestAlias bool, bidRequest *openrtb2.BidRequest) {
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
		bidRequest.Site = fpdToApply.Site
	}

	if fpdToApply.App != nil {
		bidRequest.App = fpdToApply.App
	}

	if fpdToApply.User != nil {
		//BuyerUID is a value obtained between fpd extraction and fpd application.
		//BuyerUID needs to be set back to fpd before applying this fpd to final bidder request
		if bidRequest.User != nil && len(bidRequest.User.BuyerUID) > 0 {
			fpdToApply.User.BuyerUID = bidRequest.User.BuyerUID
		}
		bidRequest.User = fpdToApply.User
	}
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

func applyBidAdjustmentToFloor(req *openrtb2.BidRequest, bidder string, adjustmentFactors map[string]float64) {
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
