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

	overrides, messages, err := findImpressionOverrides(payload, actionOverrides, btype, checkAttrExistence)
	result.Warnings = mergeStrings(result.Warnings, messages...)
	if err != nil {
		return fmt.Errorf("failed to get override for imp.*.banner.btype: %s", err)
	}

	// Filter to only apply to impressions with Banner objects
	if len(overrides) > 0 {
		filteredOverrides := filterByMediaType(payload, overrides, func(imp openrtb2.Imp) bool {
			return imp.Banner != nil
		})
		if len(filteredOverrides) > 0 {
			mutation := createBTypeMutation(filteredOverrides)
			changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "banner", "btype")
		}
		attributes.bType = overrides
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

	bannerBattr := cfg.Attributes.Battr.BlockedBannerAttr
	bannerActionOverrides := cfg.Attributes.Battr.ActionOverrides.BlockedBannerAttr
	bannerCheckAttrExistence := func(imp openrtb2.Imp) bool {
		return imp.Banner != nil && len(imp.Banner.BAttr) > 0
	}

	bannerOverrides, bannerMessages, err := findImpressionOverrides(payload, bannerActionOverrides, bannerBattr, bannerCheckAttrExistence)
	messages = append(messages, bannerMessages...)
	if err != nil {
		return fmt.Errorf("failed to get override for imp.*.banner.battr: %s", err)
	}

	// Apply banner battr only to impressions that have Banner objects
	if len(bannerOverrides) > 0 {
		filteredBannerOverrides := filterByMediaType(payload, bannerOverrides, func(imp openrtb2.Imp) bool {
			return imp.Banner != nil
		})
		if len(filteredBannerOverrides) > 0 {
			mutation := createBAttrMutation(filteredBannerOverrides, "banner")
			changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "banner", "battr")
		}
	}

	// Video battr
	videoBattr := cfg.Attributes.Battr.BlockedVideoAttr
	videoActionOverrides := cfg.Attributes.Battr.ActionOverrides.BlockedVideoAttr
	videoCheckAttrExistence := func(imp openrtb2.Imp) bool {
		return imp.Video != nil && len(imp.Video.BAttr) > 0
	}

	videoOverrides, videoMessages, err := findImpressionOverrides(payload, videoActionOverrides, videoBattr, videoCheckAttrExistence)
	messages = append(messages, videoMessages...)
	if err != nil {
		return fmt.Errorf("failed to get override for imp.*.video.battr: %s", err)
	}

	// Apply video battr only to impressions that have Video objects
	if len(videoOverrides) > 0 {
		filteredVideoOverrides := filterByMediaType(payload, videoOverrides, func(imp openrtb2.Imp) bool {
			return imp.Video != nil
		})
		if len(filteredVideoOverrides) > 0 {
			mutation := createBAttrMutation(filteredVideoOverrides, "video")
			changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "video", "battr")
		}
	}

	// Audio battr
	audioBattr := cfg.Attributes.Battr.BlockedAudioAttr
	audioActionOverrides := cfg.Attributes.Battr.ActionOverrides.BlockedAudioAttr
	audioCheckAttrExistence := func(imp openrtb2.Imp) bool {
		return imp.Audio != nil && len(imp.Audio.BAttr) > 0
	}

	audioOverrides, audioMessages, err := findImpressionOverrides(payload, audioActionOverrides, audioBattr, audioCheckAttrExistence)
	messages = append(messages, audioMessages...)
	if err != nil {
		return fmt.Errorf("failed to get override for imp.*.audio.battr: %s", err)
	}

	// Apply audio battr only to impressions that have Audio objects
	if len(audioOverrides) > 0 {
		filteredAudioOverrides := filterByMediaType(payload, audioOverrides, func(imp openrtb2.Imp) bool {
			return imp.Audio != nil
		})
		if len(filteredAudioOverrides) > 0 {
			mutation := createBAttrMutation(filteredAudioOverrides, "audio")
			changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "imp", "audio", "battr")
		}
	}

	// Store all attributes and merge messages
	attributes.bAttr = make(map[string][]int)
	for k, v := range bannerOverrides {
		attributes.bAttr[k] = v
	}
	for k, v := range videoOverrides {
		attributes.bAttr[k] = v
	}
	for k, v := range audioOverrides {
		attributes.bAttr[k] = v
	}

	result.Warnings = mergeStrings(result.Warnings, messages...)
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

// filterByMediaType filters overrides to only include impressions with a specific media type
func filterByMediaType(
	payload hookstage.BidderRequestPayload,
	overrides map[string][]int,
	mediaTypeExists func(imp openrtb2.Imp) bool,
) map[string][]int {
	filtered := make(map[string][]int)

	for _, imp := range payload.Request.Imp {
		if values, exists := overrides[imp.ID]; exists && mediaTypeExists(imp) {
			filtered[imp.ID] = values
		}
	}

	return filtered
}

// createBAttrMutation creates a mutation function for a specific media type
func createBAttrMutation(bAttrByImp map[string][]int, mediaType string) hookstage.MutationFunc[hookstage.BidderRequestPayload] {
	return func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
		for i, imp := range payload.Request.Imp {
			if values, ok := bAttrByImp[imp.ID]; ok && len(values) > 0 {
				switch mediaType {
				case "banner":
					imp.Banner.BAttr = make([]adcom1.CreativeAttribute, len(values))
					for j, attr := range values {
						imp.Banner.BAttr[j] = adcom1.CreativeAttribute(attr)
					}
				case "video":
					imp.Video.BAttr = make([]adcom1.CreativeAttribute, len(values))
					for j, attr := range values {
						imp.Video.BAttr[j] = adcom1.CreativeAttribute(attr)
					}
				case "audio":
					imp.Audio.BAttr = make([]adcom1.CreativeAttribute, len(values))
					for j, attr := range values {
						imp.Audio.BAttr[j] = adcom1.CreativeAttribute(attr)
					}
				}
				payload.Request.Imp[i] = imp
			}
		}
		return payload, nil
	}
}

func createBTypeMutation(bTypeByImp map[string][]int) hookstage.MutationFunc[hookstage.BidderRequestPayload] {
	return func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
		for i, imp := range payload.Request.Imp {
			if values, ok := bTypeByImp[imp.ID]; ok && len(values) > 0 {
				imp.Banner.BType = make([]openrtb2.BannerAdType, len(values))
				for j, btype := range values {
					imp.Banner.BType[j] = openrtb2.BannerAdType(btype)
				}
				payload.Request.Imp[i] = imp
			}
		}
		return payload, nil
	}
}
