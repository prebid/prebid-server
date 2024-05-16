package ortb

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/stored_responses"
)

type RequestValidator struct {
	bidderMap       map[string]openrtb_ext.BidderName
	disabledBidders map[string]string
	paramsValidator openrtb_ext.BidderParamValidator
}

func NewRequestValidator(bidderMap map[string]openrtb_ext.BidderName, disabledBidders map[string]string, paramsValidator openrtb_ext.BidderParamValidator) *RequestValidator {
	return &RequestValidator{
		bidderMap:       bidderMap,
		disabledBidders: disabledBidders,
		paramsValidator: paramsValidator,
	}
}

func (rv *RequestValidator) ValidateImp(imp *openrtb_ext.ImpWrapper, index int, aliases map[string]string, hasStoredResponses bool, storedBidResponses stored_responses.ImpBidderStoredResp) []error {
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

	if err := fillAndValidateNative(imp.Native, index); err != nil {
		return []error{err}
	}

	if err := validatePmp(imp.PMP, index); err != nil {
		return []error{err}
	}

	errL := rv.validateImpExt(imp, aliases, index, hasStoredResponses, storedBidResponses)
	if len(errL) != 0 {
		return errL
	}

	return nil
}

func (rv *RequestValidator) validateImpExt(imp *openrtb_ext.ImpWrapper, aliases map[string]string, impIndex int, hasStoredResponses bool, storedBidResp stored_responses.ImpBidderStoredResp) []error {
	if len(imp.Ext) == 0 {
		return []error{fmt.Errorf("request.imp[%d].ext is required", impIndex)}
	}

	impExt, err := imp.GetImpExt()
	if err != nil {
		return []error{err}
	}

	prebid := impExt.GetOrCreatePrebid()
	prebidModified := false

	if prebid.Bidder == nil {
		prebid.Bidder = make(map[string]json.RawMessage)
	}

	ext := impExt.GetExt()
	extModified := false

	// promote imp[].ext.BIDDER to newer imp[].ext.prebid.bidder.BIDDER location, with the later taking precedence
	for k, v := range ext {
		if isPossibleBidder(k) {
			if _, exists := prebid.Bidder[k]; !exists {
				prebid.Bidder[k] = v
				prebidModified = true
			}
			delete(ext, k)
			extModified = true
		}
	}

	if hasStoredResponses && prebid.StoredAuctionResponse == nil {
		return []error{fmt.Errorf("request validation failed. The StoredAuctionResponse.ID field must be completely present with, or completely absent from, all impressions in request. No StoredAuctionResponse data found for request.imp[%d].ext.prebid \n", impIndex)}
	}

	if err := rv.validateStoredBidResponses(prebid, storedBidResp, imp.ID); err != nil {
		return []error{err}
	}

	errL := []error{}

	for bidder, ext := range prebid.Bidder {
		coreBidder, _ := openrtb_ext.NormalizeBidderName(bidder)
		if tmp, isAlias := aliases[bidder]; isAlias {
			coreBidder = openrtb_ext.BidderName(tmp)
		}

		if coreBidderNormalized, isValid := rv.bidderMap[coreBidder.String()]; isValid {
			if err := rv.paramsValidator.Validate(coreBidderNormalized, ext); err != nil {
				return []error{fmt.Errorf("request.imp[%d].ext.prebid.bidder.%s failed validation.\n%v", impIndex, bidder, err)}
			}
		} else {
			if msg, isDisabled := rv.disabledBidders[bidder]; isDisabled {
				errL = append(errL, &errortypes.BidderTemporarilyDisabled{Message: msg})
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

func (rv *RequestValidator) validateStoredBidResponses(prebid *openrtb_ext.ExtImpPrebid, storedBidResp stored_responses.ImpBidderStoredResp, impId string) error {
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

// TODO: move this outside of this package
// isPossibleBidder determines if a bidder name is a potential real bidder.
func isPossibleBidder(bidder string) bool {
	switch openrtb_ext.BidderName(bidder) {
	case openrtb_ext.BidderReservedContext:
		return false
	case openrtb_ext.BidderReservedData:
		return false
	case openrtb_ext.BidderReservedGPID:
		return false
	case openrtb_ext.BidderReservedPrebid:
		return false
	case openrtb_ext.BidderReservedSKAdN:
		return false
	case openrtb_ext.BidderReservedTID:
		return false
	case openrtb_ext.BidderReservedAE:
		return false
	default:
		return true
	}
}
