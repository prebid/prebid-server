package ortb2blocking

import (
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

func handleRawBidderResponseHook(
	cfg config,
	payload hookstage.RawBidderResponsePayload,
	moduleCtx hookstage.ModuleContext,
) (result hookstage.HookResult[hookstage.RawBidderResponsePayload], err error) {
	bidder := payload.Bidder
	blockAttrsVal, ok := moduleCtx[bidder]
	if !ok {
		// if there are no blocking attributes for this bidder just pass empty blockingAttributes for further processing
		// other values from config must still be checked
		blockAttrsVal = blockingAttributes{}
	}

	blockAttrs, ok := blockAttrsVal.(blockingAttributes)
	if !ok {
		return result, hookexecution.NewFailure("could not cast blocking attributes for bidder `%s`, module context has incorrect data", payload.Bidder)
	}

	result.AnalyticsTags = newEnforceBlockingTags()

	// allowedBids will store all bids that have passed the attribute check
	allowedBids := make([]*adapters.TypedBid, 0)
	for _, bid := range payload.BidderResponse.Bids {

		failedChecksData := make(map[string]interface{})
		bidMediaTypes := mediaTypesFromBid(bid)
		dealID := bid.Bid.DealID

		failedChecksData, err = shouldBeBlockedDueToBadv(&result, cfg, blockAttrs, bid.Bid, bidMediaTypes, bidder, dealID, failedChecksData)
		if err != nil {
			addFailedStatusTag(&result)
			return result, hookexecution.NewFailure("failed to process badv block checking: %s", err)
		}

		failedChecksData, enforceBlocks, err := shouldBeBlockedDueToBcat(&result, cfg, blockAttrs, bid.Bid, bidMediaTypes, bidder, dealID, failedChecksData)
		if err != nil {
			addFailedStatusTag(&result)
			return result, hookexecution.NewFailure("failed to process bcat block checking: %s", err)
		}

		failedChecksData = shouldBeBlockedDueToCattax(blockAttrs, bid.Bid, enforceBlocks, failedChecksData)

		failedChecksData, err = shouldBeBlockedDueToBapp(&result, cfg, blockAttrs, bid.Bid, bidMediaTypes, bidder, dealID, failedChecksData)
		if err != nil {
			addFailedStatusTag(&result)
			return result, hookexecution.NewFailure("failed to process bapp block checking: %s", err)
		}

		failedChecksData, err = shouldBeBlockedDueToBattr(&result, cfg, blockAttrs, bid.Bid, bidMediaTypes, bidder, dealID, failedChecksData)
		if err != nil {
			addFailedStatusTag(&result)
			return result, hookexecution.NewFailure("failed to process battr block checking: %s", err)
		}

		if len(failedChecksData) == 0 {
			addAllowedAnalyticTag(&result, bidder, bid.Bid.ImpID)
			allowedBids = append(allowedBids, bid)
		} else {
			failedAttributes := getFailedAttributes(failedChecksData)
			addBlockedAnalyticTag(&result, bidder, bid.Bid.ImpID, failedAttributes, failedChecksData)
			addDebugMessage(&result, bid.Bid, bidder, failedAttributes)
		}
	}

	changeSet := hookstage.ChangeSet[hookstage.RawBidderResponsePayload]{}
	if len(payload.BidderResponse.Bids) != len(allowedBids) {
		changeSet.RawBidderResponse().Bids().UpdateBids(allowedBids)
		result.ChangeSet = changeSet
	}

	return result, err
}

func mediaTypesFromBid(bid *adapters.TypedBid) mediaTypes {
	return mediaTypes{string(bid.BidType): struct{}{}}
}

func shouldBeBlockedDueToBadv(
	result *hookstage.HookResult[hookstage.RawBidderResponsePayload],
	cfg config,
	blockAttr blockingAttributes,
	bid *openrtb2.Bid,
	bidMediaTypes mediaTypes,
	bidder string,
	dealID string,
	failedChecksData map[string]interface{},
) (map[string]interface{}, error) {
	badv := cfg.Attributes.Badv

	enforceBlocks, message, err := firstOrDefaultOverride(bidder, bidMediaTypes, getIsActive, badv.ActionOverrides.EnforceBlocks, badv.EnforceBlocks)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return failedChecksData, fmt.Errorf("failed to get override for badv.enforce_blocks: %s", err)
	}
	if !enforceBlocks {
		return failedChecksData, nil
	}

	blockUnknown, message, err := firstOrDefaultOverride(bidder, bidMediaTypes, getIsActive, badv.ActionOverrides.BlockUnknownAdomain, badv.BlockUnknownAdomain)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return failedChecksData, fmt.Errorf("failed to get override for badv.block_unknown_adomain: %s", err)
	}

	var dealExceptions []string
	if dealID != "" {
		dealExceptions, err = findDealExceptionsOverridesOrDefault(dealID, getNames, badv.ActionOverrides.AllowedAdomainForDeals, badv.AllowedAdomainForDeals)
		if err != nil {
			return failedChecksData, fmt.Errorf("failed to get override for badv.allowed_adomain_for_deals: %s", err)
		}
	}

	shouldBlock, blockedAttributes := blockAttribute(bid.ADomain, blockAttr.bAdv, dealExceptions, blockUnknown)
	if shouldBlock {
		failedChecksData["badv"] = blockedAttributes
	}

	return failedChecksData, nil
}

func shouldBeBlockedDueToBcat(
	result *hookstage.HookResult[hookstage.RawBidderResponsePayload],
	cfg config,
	blockAttr blockingAttributes,
	bid *openrtb2.Bid,
	bidMediaTypes mediaTypes,
	bidder, dealID string,
	failedChecksData map[string]interface{},
) (map[string]interface{}, bool, error) {
	bcat := cfg.Attributes.Bcat

	enforceBlocks, message, err := firstOrDefaultOverride(bidder, bidMediaTypes, getIsActive, bcat.ActionOverrides.EnforceBlocks, bcat.EnforceBlocks)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return failedChecksData, false, fmt.Errorf("failed to get override for bcat.enforce_blocks: %s", err)
	}
	if !enforceBlocks {
		return failedChecksData, false, nil
	}

	blockUnknown, message, err := firstOrDefaultOverride(bidder, bidMediaTypes, getIsActive, bcat.ActionOverrides.BlockUnknownAdvCat, bcat.BlockUnknownAdvCat)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return failedChecksData, enforceBlocks, fmt.Errorf("failed to get override for bcat.block_unknown_adv_cat: %s", err)
	}

	var dealExceptions []string
	if dealID != "" {
		dealExceptions, err = findDealExceptionsOverridesOrDefault(dealID, getNames, bcat.ActionOverrides.AllowedAdvCatForDeals, bcat.AllowedAdvCatForDeals)
		if err != nil {
			return failedChecksData, enforceBlocks, fmt.Errorf("failed to get override for bcat.allowed_adv_cat_for_deals: %s", err)
		}
	}

	shouldBlock, blockedAttributes := blockAttribute(bid.Cat, blockAttr.bCat, dealExceptions, blockUnknown)
	if shouldBlock {
		failedChecksData["bcat"] = blockedAttributes
	}

	return failedChecksData, enforceBlocks, nil
}

func shouldBeBlockedDueToCattax(
	blockAttr blockingAttributes,
	bid *openrtb2.Bid,
	enforceBlocks bool,
	failedChecksData map[string]interface{},
) map[string]interface{} {
	if !enforceBlocks {
		return failedChecksData
	}

	cattax := bid.CatTax
	if cattax == 0 {
		return failedChecksData
	}

	// if blocking cattax was not specified use a default value
	if blockAttr.catTax == 0 {
		blockAttr.catTax = adcom1.CatTaxIABContent10
	}

	// cattax check has a reverse logic, the blockAttr.cattax should have one allowed value
	if cattax != blockAttr.catTax {
		failedChecksData["cattax"] = []adcom1.CategoryTaxonomy{cattax}
	}

	return failedChecksData
}

func shouldBeBlockedDueToBapp(
	result *hookstage.HookResult[hookstage.RawBidderResponsePayload],
	cfg config,
	blockAttr blockingAttributes,
	bid *openrtb2.Bid,
	bidMediaTypes mediaTypes,
	bidder, dealID string,
	failedChecksData map[string]interface{},
) (map[string]interface{}, error) {
	bapp := cfg.Attributes.Bapp

	enforceBlocks, message, err := firstOrDefaultOverride(bidder, bidMediaTypes, getIsActive, bapp.ActionOverrides.EnforceBlocks, bapp.EnforceBlocks)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return failedChecksData, fmt.Errorf("failed to get override for bapp.enforce_blocks: %s", err)
	}
	if !enforceBlocks {
		return failedChecksData, nil
	}

	var dealExceptions []string
	if dealID != "" {
		dealExceptions, err = findDealExceptionsOverridesOrDefault(dealID, getNames, bapp.ActionOverrides.AllowedAppForDeals, bapp.AllowedAppForDeals)
		if err != nil {
			return failedChecksData, fmt.Errorf("failed to get override for bapp.allowed_app_for_deals: %s", err)
		}
	}

	bidBapp := []string{bid.Bundle}

	shouldBlock, blockedAttributes := blockAttribute(bidBapp, blockAttr.bApp, dealExceptions, false)
	if shouldBlock {
		failedChecksData["bapp"] = blockedAttributes
	}

	return failedChecksData, nil
}

func shouldBeBlockedDueToBattr(
	result *hookstage.HookResult[hookstage.RawBidderResponsePayload],
	cfg config,
	blockAttr blockingAttributes,
	bid *openrtb2.Bid,
	bidMediaTypes mediaTypes,
	bidder, dealID string,
	failedChecksData map[string]interface{},
) (map[string]interface{}, error) {
	battr := cfg.Attributes.Battr

	enforceBlocks, message, err := firstOrDefaultOverride(bidder, bidMediaTypes, getIsActive, battr.ActionOverrides.EnforceBlocks, battr.EnforceBlocks)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return failedChecksData, fmt.Errorf("failed to get override for battr.enforce_blocks: %s", err)
	}
	if !enforceBlocks {
		return failedChecksData, nil
	}

	var dealExceptions []int
	if dealID != "" {
		dealExceptions, err = findDealExceptionsOverridesOrDefault(dealID, getIds, battr.ActionOverrides.AllowedBannerAttrForDeals, battr.AllowedBannerAttrForDeals)
		if err != nil {
			return failedChecksData, fmt.Errorf("failed to get override for battr.allowed_banner_attr_for_deals: %s", err)
		}
	}

	if blockAttr.bAttr == nil || len(blockAttr.bAttr) == 0 {
		return failedChecksData, nil
	}
	blockedBattr := blockAttr.bAttr[bid.ImpID]
	bidAttr := toInt(bid.Attr)

	shouldBlock, blockedAttributes := blockAttribute(bidAttr, blockedBattr, dealExceptions, false)
	if shouldBlock {
		failedChecksData["battr"] = blockedAttributes
	}

	return failedChecksData, nil
}

func blockAttribute[Attribute comparable](attributes, blockedAttributes, dealExceptions []Attribute, blockUnknown bool) (bool, []Attribute) {
	if len(attributes) == 0 {
		if blockUnknown {
			return true, nil
		}
		return false, nil
	}

	if len(blockedAttributes) == 0 {
		return false, nil
	}

	blockedElements := make([]Attribute, 0)
	shouldBlock := true
	for _, attribute := range attributes {
		for _, blockedAttribute := range blockedAttributes {
			if attribute == blockedAttribute {
				for _, allowedAttribute := range dealExceptions {
					if attribute == allowedAttribute {
						shouldBlock = false
						break
					}
				}
				if shouldBlock {
					blockedElements = append(blockedElements, attribute)
				}
			}
			shouldBlock = true
		}
	}

	if len(blockedElements) == 0 {
		return false, nil
	}

	return true, blockedElements
}

func findDealExceptionsOverridesOrDefault[T any](
	dealID string,
	overrideGetter overrideGetterFn[[]T],
	actionOverrides []ActionOverride,
	defaultOverride []T,
) (overrides []T, err error) {
	for _, action := range actionOverrides {
		if action.Conditions.DealIds == nil {
			return overrides, errors.New("conditions field in account configuration must contain deal_ids")
		}

		for _, id := range action.Conditions.DealIds {
			if id == dealID {
				actionOverride, err := overrideGetter(action.Override)
				if err != nil {
					return overrides, err
				}
				overrides = append(overrides, actionOverride...)
			}
		}
	}

	if len(overrides) == 0 {
		return defaultOverride, nil
	}

	overrides = append(overrides, defaultOverride...)

	return overrides, nil
}

func addDebugMessage(
	result *hookstage.HookResult[hookstage.RawBidderResponsePayload],
	bid *openrtb2.Bid,
	bidder string,
	failedAttributes []string,
) {
	result.DebugMessages = append(
		result.DebugMessages,
		fmt.Sprintf("Bid %s from bidder %s has been rejected, failed checks: %s", bid.ID, bidder, strings.Join(failedAttributes, ", ")),
	)
}

// returns a slice with the names of failed attributes
func getFailedAttributes(data map[string]interface{}) []string {
	var builder []string
	for _, attribute := range [5]string{
		"badv",
		"bcat",
		"cattax",
		"bapp",
		"battr",
	} {
		if _, ok := data[attribute]; ok {
			builder = append(builder, attribute)
		}
	}

	return builder
}
