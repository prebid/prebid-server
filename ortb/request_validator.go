package ortb

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/stored_responses"
)

type ValidationConfig struct {
	SkipBidderParams bool
	SkipNative       bool
}

type RequestValidator interface {
	ValidateImp(imp *openrtb_ext.ImpWrapper, cfg ValidationConfig, index int, aliases map[string]string, hasStoredAuctionResponses bool, storedBidResponses stored_responses.ImpBidderStoredResp) []error
}

func NewRequestValidator(bidderMap map[string]openrtb_ext.BidderName, disabledBidders map[string]string, paramsValidator openrtb_ext.BidderParamValidator) RequestValidator {
	return &standardRequestValidator{
		bidderMap:       bidderMap,
		disabledBidders: disabledBidders,
		paramsValidator: paramsValidator,
	}
}

type standardRequestValidator struct {
	bidderMap       map[string]openrtb_ext.BidderName
	disabledBidders map[string]string
	paramsValidator openrtb_ext.BidderParamValidator
}

func (srv *standardRequestValidator) ValidateImp(imp *openrtb_ext.ImpWrapper, cfg ValidationConfig, index int, aliases map[string]string, hasStoredAuctionResponses bool, storedBidResponses stored_responses.ImpBidderStoredResp) []error {
	if imp.ID == "" {
		return []error{fmt.Errorf("request.imp[%d] missing required field: \"id\"", index)}
	}

	if len(imp.Metric) != 0 {
		return []error{fmt.Errorf("request.imp[%d].metric is not yet supported by prebid-server. Support may be added in the future", index)}
	}

	if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
		return []error{fmt.Errorf("request.imp[%d] must contain at least one of \"banner\", \"video\", \"audio\", or \"native\"", index)}
	}

	if err := validateBanner(imp.Banner, index, isInterstitial(imp)); err != nil {
		return []error{err}
	}

	if err := validateVideo(imp.Video, index); err != nil {
		return []error{err}
	}

	if err := validateAudio(imp.Audio, index); err != nil {
		return []error{err}
	}

	if !cfg.SkipNative {
		if err := fillAndValidateNative(imp.Native, index); err != nil {
			return []error{err}
		}
	}

	if err := validatePmp(imp.PMP, index); err != nil {
		return []error{err}
	}

	errL := srv.validateImpExt(imp, cfg, aliases, index, hasStoredAuctionResponses, storedBidResponses)
	if len(errL) != 0 {
		return errL
	}

	return nil
}

func (srv *standardRequestValidator) validateImpExt(imp *openrtb_ext.ImpWrapper, cfg ValidationConfig, aliases map[string]string, impIndex int, hasStoredAuctionResponses bool, storedBidResp stored_responses.ImpBidderStoredResp) []error {
	if len(imp.Ext) == 0 {
		return []error{fmt.Errorf("request.imp[%d].ext is required", impIndex)}
	}

	impExt, err := imp.GetImpExt()
	if err != nil {
		return []error{err}
	}

	prebid := impExt.GetOrCreatePrebid()
	prebidModified := false

	bidderPromote := false

	if prebid.Bidder == nil {
		prebid.Bidder = make(map[string]json.RawMessage)
		bidderPromote = true
	}

	ext := impExt.GetExt()
	extModified := false

	// promote imp[].ext.BIDDER to newer imp[].ext.prebid.bidder.BIDDER location, with the later taking precedence
	if bidderPromote {
		for k, v := range ext {
			if openrtb_ext.IsPotentialBidder(k) {
				if _, exists := prebid.Bidder[k]; !exists {
					prebid.Bidder[k] = v
					prebidModified = true
				}
				delete(ext, k)
				extModified = true
			}
		}
	}

	if hasStoredAuctionResponses && prebid.StoredAuctionResponse == nil {
		return []error{fmt.Errorf("request validation failed. The StoredAuctionResponse.ID field must be completely present with, or completely absent from, all impressions in request. No StoredAuctionResponse data found for request.imp[%d].ext.prebid \n", impIndex)}
	}

	if err := srv.validateStoredBidResponses(prebid, storedBidResp, imp.ID); err != nil {
		return []error{err}
	}

	errL := []error{}

	for bidder, val := range prebid.Bidder {
		coreBidder, _ := openrtb_ext.NormalizeBidderName(bidder)
		if tmp, isAlias := aliases[bidder]; isAlias {
			coreBidder = openrtb_ext.BidderName(tmp)
		}

		if coreBidderNormalized, isValid := srv.bidderMap[coreBidder.String()]; isValid {
			if !cfg.SkipBidderParams {
				if err := srv.paramsValidator.Validate(coreBidderNormalized, val); err != nil {
					return []error{fmt.Errorf("request.imp[%d].ext.prebid.bidder.%s failed validation.\n%v", impIndex, bidder, err)}
				}
			}
		} else {
			if msg, isDisabled := srv.disabledBidders[bidder]; isDisabled {
				errL = append(errL, &errortypes.BidderTemporarilyDisabled{Message: msg})
				delete(prebid.Bidder, bidder)
				prebidModified = true
			} else if bidderPromote {
				errL = append(errL, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d].ext contains unknown bidder: '%s', ignoring", impIndex, bidder)})
				ext[bidder] = val
				delete(prebid.Bidder, bidder)
				prebidModified = true
			} else {
				return []error{fmt.Errorf("request.imp[%d].ext.prebid.bidder contains unknown bidder: %s. Did you forget an alias in request.ext.prebid.aliases?", impIndex, bidder)}
			}
		}
	}

	if len(prebid.Bidder) == 0 {
		errL = append(errL, fmt.Errorf("request.imp[%d].ext.prebid.bidder must contain at least one bidder", impIndex))
		return errL
	}

	if prebidModified {
		impExt.SetPrebid(prebid)
	}
	if extModified {
		impExt.SetExt(ext)
	}

	return errL
}

func (srv *standardRequestValidator) validateStoredBidResponses(prebid *openrtb_ext.ExtImpPrebid, storedBidResp stored_responses.ImpBidderStoredResp, impId string) error {
	if storedBidResp == nil && len(prebid.StoredBidResponse) == 0 {
		return nil
	}

	if storedBidResp == nil {
		return generateStoredBidResponseValidationError(impId)
	}
	if bidResponses, ok := storedBidResp[impId]; ok {
		if len(bidResponses) != len(prebid.Bidder) {
			return generateStoredBidResponseValidationError(impId)
		}

		for bidderName := range bidResponses {
			if _, bidderNameOk := openrtb_ext.NormalizeBidderName(bidderName); !bidderNameOk {
				return fmt.Errorf(`unrecognized bidder "%v"`, bidderName)
			}
			if _, present := prebid.Bidder[bidderName]; !present {
				return generateStoredBidResponseValidationError(impId)
			}
		}
	}
	return nil
}

func generateStoredBidResponseValidationError(impID string) error {
	return fmt.Errorf("request validation failed. Stored bid responses are specified for imp %s. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse", impID)
}
