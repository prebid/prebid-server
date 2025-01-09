package ortb2blocking

import (
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func handleBidderRequestHook(
	cfg config,
	payload hookstage.BidderRequestPayload,
) (result hookstage.HookResult[hookstage.BidderRequestPayload], err error) {
	if payload.Request == nil || payload.Request.BidRequest == nil {
		return result, hookexecution.NewFailure("payload contains a nil bid request")
	}

	mediaTypes := mediaTypesFrom(payload.Request.BidRequest)
	changeSet := hookstage.ChangeSet[hookstage.BidderRequestPayload]{}
	blockingAttributes := blockingAttributes{}

	if err = updateBAdv(cfg, payload, mediaTypes, &blockingAttributes, &result, &changeSet); err != nil {
		return result, hookexecution.NewFailure("failed to update badv field: %s", err)
	}

	if err = updateBApp(cfg, payload, mediaTypes, &blockingAttributes, &result, &changeSet); err != nil {
		return result, hookexecution.NewFailure("failed to update bapp field: %s", err)
	}

	if err = updateBCat(cfg, payload, mediaTypes, &blockingAttributes, &result, &changeSet); err != nil {
		return result, hookexecution.NewFailure("failed to update bcat field: %s", err)
	}

	if err = updateBType(cfg, payload, &blockingAttributes, &result, &changeSet); err != nil {
		return result, hookexecution.NewFailure("failed to update btype field: %s", err)
	}

	if err = updateBAttr(cfg, payload, &blockingAttributes, &result, &changeSet); err != nil {
		return result, hookexecution.NewFailure("failed to update battr field: %s", err)
	}

	updateCatTax(cfg, payload, &blockingAttributes, &changeSet)

	result.ChangeSet = changeSet
	result.ModuleContext = hookstage.ModuleContext{payload.Bidder: blockingAttributes}

	return result, nil
}

func updateBAdv(
	cfg config,
	payload hookstage.BidderRequestPayload,
	mediaTypes mediaTypes,
	attributes *blockingAttributes,
	result *hookstage.HookResult[hookstage.BidderRequestPayload],
	changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload],
) (err error) {
	if len(payload.Request.BAdv) > 0 {
		return nil
	}

	var message string
	badv := cfg.Attributes.Badv.BlockedAdomain
	actionOverrides := cfg.Attributes.Badv.ActionOverrides.BlockedAdomain

	attributes.bAdv, message, err = firstOrDefaultOverride(payload.Bidder, mediaTypes, getNames, actionOverrides, badv)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return fmt.Errorf("failed to get override for badv.blocked_adomain: %s", err)
	} else if len(attributes.bAdv) > 0 {
		changeSet.BidderRequest().BAdv().Update(attributes.bAdv)
	}

	return nil
}

func updateBApp(
	cfg config,
	payload hookstage.BidderRequestPayload,
	mediaTypes mediaTypes,
	attributes *blockingAttributes,
	result *hookstage.HookResult[hookstage.BidderRequestPayload],
	changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload],
) (err error) {
	if len(payload.Request.BApp) > 0 {
		return nil
	}

	var message string
	bapp := cfg.Attributes.Bapp.BlockedApp
	actionOverrides := cfg.Attributes.Bapp.ActionOverrides.BlockedApp

	attributes.bApp, message, err = firstOrDefaultOverride(payload.Bidder, mediaTypes, getNames, actionOverrides, bapp)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return fmt.Errorf("failed to get override for bapp.blocked_app: %s", err)
	} else if len(attributes.bApp) > 0 {
		changeSet.BidderRequest().BApp().Update(attributes.bApp)
	}

	return nil
}

func updateBCat(
	cfg config,
	payload hookstage.BidderRequestPayload,
	mediaTypes mediaTypes,
	attributes *blockingAttributes,
	result *hookstage.HookResult[hookstage.BidderRequestPayload],
	changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload],
) (err error) {
	if len(payload.Request.BCat) > 0 {
		return nil
	}

	var message string
	bcat := cfg.Attributes.Bcat.BlockedAdvCat
	actionOverrides := cfg.Attributes.Bcat.ActionOverrides.BlockedAdvCat

	attributes.bCat, message, err = firstOrDefaultOverride(payload.Bidder, mediaTypes, getNames, actionOverrides, bcat)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return fmt.Errorf("failed to get override for bcat.blocked_adv_cat: %s", err)
	} else if len(attributes.bCat) > 0 {
		changeSet.BidderRequest().BCat().Update(attributes.bCat)
	}

	return nil
}

func updateBType(
	cfg config,
	payload hookstage.BidderRequestPayload,
	attributes *blockingAttributes,
	result *hookstage.HookResult[hookstage.BidderRequestPayload],
	changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload],
) (err error) {
	var messages []string
	btype := cfg.Attributes.Btype.BlockedBannerType
	actionOverrides := cfg.Attributes.Btype.ActionOverrides.BlockedBannerType
	checkAttrExistence := func(imp openrtb2.Imp) bool {
		return imp.Banner != nil && len(imp.Banner.BType) > 0
	}

	attributes.bType, messages, err = findImpressionOverrides(payload, actionOverrides, btype, checkAttrExistence)
	result.Warnings = mergeStrings(result.Warnings, messages...)
	if err != nil {
		return fmt.Errorf("failed to get override for imp.*.banner.btype: %s", err)
	} else if len(attributes.bType) > 0 {
		mutation := bTypeMutation(attributes.bType)
		changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "banner", "btype")
	}

	return nil
}

func updateBAttr(
	cfg config,
	payload hookstage.BidderRequestPayload,
	attributes *blockingAttributes,
	result *hookstage.HookResult[hookstage.BidderRequestPayload],
	changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload],
) (err error) {
	var messages []string
	battr := cfg.Attributes.Battr.BlockedBannerAttr
	actionOverrides := cfg.Attributes.Battr.ActionOverrides.BlockedBannerAttr
	checkAttrExistence := func(imp openrtb2.Imp) bool {
		return imp.Banner != nil && len(imp.Banner.BAttr) > 0
	}

	attributes.bAttr, messages, err = findImpressionOverrides(payload, actionOverrides, battr, checkAttrExistence)
	result.Warnings = mergeStrings(result.Warnings, messages...)
	if err != nil {
		return fmt.Errorf("failed to get override for imp.*.banner.battr: %s", err)
	} else if len(attributes.bAttr) > 0 {
		mutation := bAttrMutation(attributes.bAttr)
		changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "banner", "battr")
	}

	return nil
}

func updateCatTax(
	cfg config,
	payload hookstage.BidderRequestPayload,
	attributes *blockingAttributes,
	changeSet *hookstage.ChangeSet[hookstage.BidderRequestPayload],
) {
	if payload.Request.CatTax > 0 {
		return
	}

	attributes.catTax = cfg.Attributes.Bcat.CategoryTaxonomy
	changeSet.BidderRequest().CatTax().Update(attributes.catTax)
}

func bTypeMutation(bTypeByImp map[string][]int) hookstage.MutationFunc[hookstage.BidderRequestPayload] {
	return mutationForImp(bTypeByImp, func(imp openrtb2.Imp, btype []int) openrtb2.Imp {
		imp.Banner.BType = make([]openrtb2.BannerAdType, len(btype))
		for i := range btype {
			imp.Banner.BType[i] = openrtb2.BannerAdType(btype[i])
		}
		return imp
	})
}

func bAttrMutation(bAttrByImp map[string][]int) hookstage.MutationFunc[hookstage.BidderRequestPayload] {
	return mutationForImp(bAttrByImp, func(imp openrtb2.Imp, battr []int) openrtb2.Imp {
		imp.Banner.BAttr = make([]adcom1.CreativeAttribute, len(battr))
		for i := range battr {
			imp.Banner.BAttr[i] = adcom1.CreativeAttribute(battr[i])
		}
		return imp
	})
}

type impUpdateFunc func(imp openrtb2.Imp, values []int) openrtb2.Imp

func mutationForImp(
	valuesByImp map[string][]int,
	impUpdater impUpdateFunc,
) hookstage.MutationFunc[hookstage.BidderRequestPayload] {
	return func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
		for i, imp := range payload.Request.Imp {
			if values, ok := valuesByImp[imp.ID]; ok {
				if len(values) == 0 {
					continue
				}

				if imp.Banner == nil {
					imp.Banner = &openrtb2.Banner{}
				}

				payload.Request.Imp[i] = impUpdater(imp, values)
			}
		}
		return payload, nil
	}
}

// firstOrDefaultOverride searches for matching override based on conditions.
// Returns only first found override. Override for specific bidder has higher priority
// than override matching all bidders. If no override found, the defaultOverride returned.
func firstOrDefaultOverride[T any](
	bidder string,
	requestMediaTypes mediaTypes,
	overrideGetter overrideGetterFn[T],
	actionOverrides []ActionOverride,
	defaultOverride T,
) (override T, message string, err error) {
	var allOverrides []T
	var specificOverrides []T

	for _, action := range actionOverrides {
		if err = validateCondition(action.Conditions); err != nil {
			return override, message, err
		}

		matchAllBidders := action.Conditions.Bidders == nil
		matchesBidder := matchAllBidders || hasMatches(action.Conditions.Bidders, bidder)
		matchesMedia := action.Conditions.MediaTypes == nil || requestMediaTypes.intersects(action.Conditions.MediaTypes)

		if matchesBidder && matchesMedia {
			actionOverride, err := overrideGetter(action.Override)
			if err != nil {
				return override, message, err
			}

			if matchAllBidders {
				allOverrides = append(allOverrides, actionOverride)
			} else {
				specificOverrides = append(specificOverrides, actionOverride)
			}
		}
	}

	if len(specificOverrides)+len(allOverrides) > 1 {
		message = fmt.Sprintf(
			"More than one condition matches request. Bidder: %s, request media types: %s",
			bidder,
			requestMediaTypes,
		)
	}

	if len(specificOverrides) > 0 {
		override = specificOverrides[0]
	} else if len(allOverrides) > 0 {
		override = allOverrides[0]
	} else {
		override = defaultOverride
	}

	return override, message, nil
}

// findImpressionOverrides returns overrides for each of the BidRequest impressions.
// Overrides returned in format map[ImpressionID][]int.
func findImpressionOverrides(
	payload hookstage.BidderRequestPayload,
	actionOverrides []ActionOverride,
	defaultOverride []int,
	isAttrPresent func(imp openrtb2.Imp) bool,
) (map[string][]int, []string, error) {
	bidder := payload.Bidder
	overrides := map[string][]int{}
	messages := []string{}

	for _, imp := range payload.Request.Imp {
		// do not add override for attribute if it already exists in request
		if isAttrPresent(imp) {
			continue
		}

		mediaTypes := mediaTypesFromImp(imp)
		override, message, err := firstOrDefaultOverride(bidder, mediaTypes, getIds, actionOverrides, defaultOverride)
		messages = mergeStrings(messages, message)
		if err != nil {
			return nil, messages, err
		} else if len(override) > 0 {
			overrides[imp.ID] = override
		}
	}

	return overrides, messages, nil
}

type overrideGetterFn[T any] func(override Override) (T, error)

func getNames(override Override) ([]string, error) {
	if len(override.Names) == 0 {
		return nil, errors.New("empty override field")
	}
	return override.Names, nil
}

func getIds(override Override) ([]int, error) {
	if len(override.Ids) == 0 {
		return nil, errors.New("empty override field")
	}
	return override.Ids, nil
}

func getIsActive(override Override) (bool, error) {
	return override.IsActive, nil
}

type mediaTypes map[string]struct{}

func (m mediaTypes) String() string {
	var builder strings.Builder
	// keep media_types in sorted order
	for i, mediaType := range [4]string{
		string(openrtb_ext.BidTypeAudio),
		string(openrtb_ext.BidTypeBanner),
		string(openrtb_ext.BidTypeNative),
		string(openrtb_ext.BidTypeVideo),
	} {
		if _, ok := m[mediaType]; ok {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(mediaType)
		}
	}

	return builder.String()
}

func (m mediaTypes) intersects(mediaTypes []string) bool {
	for _, mediaType := range mediaTypes {
		if _, ok := m[strings.ToLower(mediaType)]; ok {
			return true
		}
	}
	return false
}

func mediaTypesFrom(request *openrtb2.BidRequest) mediaTypes {
	mediaTypes := mediaTypes{}
	for _, imp := range request.Imp {
		if len(mediaTypes) == 4 {
			break
		}

		for mt, val := range mediaTypesFromImp(imp) {
			mediaTypes[mt] = val
		}
	}

	return mediaTypes
}

func mediaTypesFromImp(imp openrtb2.Imp) mediaTypes {
	mediaTypes := mediaTypes{}
	if imp.Audio != nil {
		mediaTypes[string(openrtb_ext.BidTypeAudio)] = struct{}{}
	}

	if imp.Banner != nil {
		mediaTypes[string(openrtb_ext.BidTypeBanner)] = struct{}{}
	}

	if imp.Native != nil {
		mediaTypes[string(openrtb_ext.BidTypeNative)] = struct{}{}
	}

	if imp.Video != nil {
		mediaTypes[string(openrtb_ext.BidTypeVideo)] = struct{}{}
	}

	return mediaTypes
}

func validateCondition(conditions Conditions) error {
	if conditions.Bidders == nil && conditions.MediaTypes == nil {
		return errors.New("bidders and media_types absent from conditions, at least one of the fields must be present")
	}
	return nil
}
