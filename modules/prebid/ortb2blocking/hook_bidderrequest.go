package ortb2blocking

import (
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v17/adcom1"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks/hookexecution"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func handleBidderRequestHook(
	cfg Config,
	payload hookstage.BidderRequestPayload,
) (result hookstage.HookResult[hookstage.BidderRequestPayload], err error) {
	if payload.BidRequest == nil {
		return result, hookexecution.NewFailure("empty BidRequest provided")
	}

	var message string
	var messages []string
	var blockingAttributes blockingAttributes

	bidder := payload.Bidder
	mediaTypes := mediaTypesFrom(payload.BidRequest)
	changeSet := hookstage.ChangeSet[hookstage.BidderRequestPayload]{}

	badv := cfg.Attributes.Badv.BlockedAdomain
	actionOverrides := cfg.Attributes.Badv.ActionOverrides.BlockedAdomain
	blockingAttributes.badv, message, err = firstOrDefaultOverride(bidder, mediaTypes, getNames, actionOverrides, badv)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return result, hookexecution.NewFailure("failed to get override for badv.blocked_adomain: %s", err)
	} else if len(badv) > 0 {
		changeSet.BidderRequest().BAdv().Update(badv)
	}

	bapp := cfg.Attributes.Bapp.BlockedApp
	actionOverrides = cfg.Attributes.Bapp.ActionOverrides.BlockedApp
	blockingAttributes.bapp, message, err = firstOrDefaultOverride(bidder, mediaTypes, getNames, actionOverrides, bapp)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return result, hookexecution.NewFailure("failed to get override for bapp.blocked_app: %s", err)
	} else if len(bapp) > 0 {
		changeSet.BidderRequest().BApp().Update(bapp)
	}

	bcat := cfg.Attributes.Bcat.BlockedAdvCat
	actionOverrides = cfg.Attributes.Bcat.ActionOverrides.BlockedAdvCat
	blockingAttributes.bcat, message, err = firstOrDefaultOverride(bidder, mediaTypes, getNames, actionOverrides, bcat)
	result.Warnings = mergeStrings(result.Warnings, message)
	if err != nil {
		return result, hookexecution.NewFailure("failed to get override for bcat.blocked_adv_cat: %s", err)
	} else if len(bcat) > 0 {
		changeSet.BidderRequest().BCat().Update(bcat)
	}

	blockingAttributes.cattax = payload.BidRequest.CatTax
	if blockingAttributes.cattax == 0 && cfg.Attributes.Bcat.CategoryTaxonomy > 0 {
		blockingAttributes.cattax = cfg.Attributes.Bcat.CategoryTaxonomy
		changeSet.BidderRequest().CatTax().Update(blockingAttributes.cattax)
	}

	btype := cfg.Attributes.Btype.BlockedBannerType
	actionOverrides = cfg.Attributes.Btype.ActionOverrides.BlockedBannerType
	blockingAttributes.btype, messages, err = findImpressionOverrides(payload, getIds, actionOverrides, btype)
	result.Warnings = mergeStrings(result.Warnings, messages...)
	if err != nil {
		return result, hookexecution.NewFailure("failed to get override for imp.*.banner.btype: %s", err)
	} else if len(blockingAttributes.btype) > 0 {
		mutation := bTypeMutation(blockingAttributes)
		changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "banner", "btype")
	}

	battr := cfg.Attributes.Battr.BlockedBannerAttr
	actionOverrides = cfg.Attributes.Battr.ActionOverrides.BlockedBannerAttr
	blockingAttributes.battr, messages, err = findImpressionOverrides(payload, getIds, actionOverrides, battr)
	result.Warnings = mergeStrings(result.Warnings, messages...)
	if err != nil {
		return result, hookexecution.NewFailure("failed to get override for imp.*.banner.btype: %s", err)
	} else if len(blockingAttributes.battr) > 0 {
		mutation := bAttrMutation(blockingAttributes)
		changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "banner", "battr")
	}

	result.ChangeSet = changeSet
	result.ModuleContext = hookstage.ModuleContext{ctxKeyBlockingAttributes: blockingAttributes}

	return result, nil
}

func bTypeMutation(attributes blockingAttributes) hookstage.MutationFunc[hookstage.BidderRequestPayload] {
	return mutationForImp(attributes.btype, func(imp openrtb2.Imp, btype []int) openrtb2.Imp {
		imp.Banner.BType = make([]openrtb2.BannerAdType, len(btype))
		for i := range btype {
			imp.Banner.BType[i] = openrtb2.BannerAdType(btype[i])
		}
		return imp
	})
}

func bAttrMutation(attributes blockingAttributes) hookstage.MutationFunc[hookstage.BidderRequestPayload] {
	return mutationForImp(attributes.battr, func(imp openrtb2.Imp, battr []int) openrtb2.Imp {
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
		for i, imp := range payload.BidRequest.Imp {
			if values, ok := valuesByImp[imp.ID]; ok {
				if len(values) == 0 {
					continue
				}

				if imp.Banner == nil {
					imp.Banner = &openrtb2.Banner{}
				}

				payload.BidRequest.Imp[i] = impUpdater(imp, values)
			}
		}
		return payload, nil
	}
}

// firstOrDefaultOverride searches for matching override based on conditions.
// Returns first found override. Override for specific bidder has higher priority
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
// Overrides returned in format map[ImpressionID]Override.
func findImpressionOverrides[T any](
	payload hookstage.BidderRequestPayload,
	overrideGetter overrideGetterFn[T],
	actionOverrides []ActionOverride,
	defaultOverride T,
) (map[string]T, []string, error) {
	bidder := payload.Bidder
	overrides := map[string]T{}
	messages := []string{}

	for _, imp := range payload.BidRequest.Imp {
		mediaTypes := mediaTypesFromImp(imp)
		override, message, err := firstOrDefaultOverride(bidder, mediaTypes, overrideGetter, actionOverrides, defaultOverride)
		messages = mergeStrings(messages, message)
		if err != nil {
			return nil, messages, err
		}

		overrides[imp.ID] = override
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
	var i int
	var builder strings.Builder
	for mType := range m {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(mType)
		i++
	}

	return builder.String()
}

func (m mediaTypes) intersects(mediaTypes []string) bool {
	for _, mt := range mediaTypes {
		if _, ok := m[strings.ToLower(mt)]; ok {
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
