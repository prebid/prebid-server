package adapters

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// InfoAwareBidder wraps a Bidder to ensure all requests abide by the capabilities and
// media types defined in the static/bidder-info/{bidder}.yaml file.
//
// It adjusts incoming requests in the following ways:
//  1. If App, Site or DOOH traffic is not supported by the info file, then requests from
//     those sources will be rejected before the delegate is called.
//  2. If a given MediaType is not supported for the platform, then it will be set
//     to nil before the request is forwarded to the delegate.
//  3. Any Imps which have no MediaTypes left will be removed.
//  4. If there are no valid Imps left, the delegate won't be called at all.
type InfoAwareBidder struct {
	Bidder
	info parsedBidderInfo
}

// BuildInfoAwareBidder wraps a bidder to enforce inventory {site, app, dooh} and media type support.
func BuildInfoAwareBidder(bidder Bidder, info config.BidderInfo) Bidder {
	return &InfoAwareBidder{
		Bidder: bidder,
		info:   parseBidderInfo(info),
	}
}

func (i *InfoAwareBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *ExtraRequestInfo) ([]*RequestData, []error) {
	var allowedMediaTypes parsedSupports

	if request.Site != nil {
		if !i.info.site.enabled {
			return nil, []error{&errortypes.Warning{Message: "this bidder does not support site requests"}}
		}
		allowedMediaTypes = i.info.site
	}
	if request.App != nil {
		if !i.info.app.enabled {
			return nil, []error{&errortypes.Warning{Message: "this bidder does not support app requests"}}
		}
		allowedMediaTypes = i.info.app
	}
	if request.DOOH != nil {
		if !i.info.dooh.enabled {
			return nil, []error{&errortypes.Warning{Message: "this bidder does not support dooh requests"}}
		}
		allowedMediaTypes = i.info.dooh
	}

	updated, updatedImps, errs := pruneImps(request.Imp, allowedMediaTypes, i.info.multiformat, reqInfo.PreferredMediaType)
	if updated {
		request.Imp = updatedImps
	}

	// If all imps in bid request are invalid, exit
	if len(request.Imp) == 0 {
		return nil, append(errs, &errortypes.Warning{Message: "Bid request didn't contain media types supported by the bidder"})
	}

	reqs, delegateErrs := i.Bidder.MakeRequests(request, reqInfo)
	return reqs, append(errs, delegateErrs...)
}

// pruneImps trims imps that don't match the allowed types and removes imps that don't have the allowed types.
// It also handles multi-format restrictions if the bidder doesn't support multi-format impressions.
func pruneImps(imps []openrtb2.Imp, allowedTypes parsedSupports, multiformatSupport bool, preferredMediaType openrtb_ext.BidType) (bool, []openrtb2.Imp, []error) {
	var updated bool
	var errs []error
	writeIndex := 0

	for i := 0; i < len(imps); i++ {
		imp := &imps[i] // Work with pointer to avoid copying

		// Prune unsupported media types
		if !allowedTypes.banner && imp.Banner != nil {
			imp.Banner = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses banner, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.video && imp.Video != nil {
			imp.Video = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses video, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.audio && imp.Audio != nil {
			imp.Audio = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses audio, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.native && imp.Native != nil {
			imp.Native = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses native, but this bidder doesn't support it", i)})
		}

		// Skip if all media types are gone
		numOfFormats := countNumberOfFormats(imp)
		if numOfFormats == 0 {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d] has no supported MediaTypes. It will be ignored", i)})
			updated = true
			continue
		}

		// Handle multi-format restrictions for bidders that don't support multi-format impressions
		if !multiformatSupport && numOfFormats > 1 {

			removeImp, multiformatErrs := adjustImpForPreferredMediaType(imp, preferredMediaType)

			//remove the Imp if the bidder doesn't support the preferred media type or preferred media type is not defined
			if removeImp {
				errs = append(errs, multiformatErrs...)
				updated = true
				continue
			}

			if multiformatErrs != nil {
				errs = append(errs, multiformatErrs...)
			}
		}

		// Move valid imp to the correct position
		if updated {
			imps[writeIndex] = imps[i]
		}
		writeIndex++
	}
	// If updated, return the modified slice; otherwise, return the original slice
	if updated {
		return true, imps[:writeIndex], errs
	}
	return false, imps, errs
}

func parseAllowedTypes(allowedTypes []openrtb_ext.BidType) (allowBanner bool, allowVideo bool, allowAudio bool, allowNative bool) {
	for _, allowedType := range allowedTypes {
		switch allowedType {
		case openrtb_ext.BidTypeBanner:
			allowBanner = true
		case openrtb_ext.BidTypeVideo:
			allowVideo = true
		case openrtb_ext.BidTypeAudio:
			allowAudio = true
		case openrtb_ext.BidTypeNative:
			allowNative = true
		}
	}
	return
}

func countNumberOfFormats(imp *openrtb2.Imp) int {
	count := 0
	if imp.Banner != nil {
		count++
	}
	if imp.Video != nil {
		count++
	}
	if imp.Audio != nil {
		count++
	}
	if imp.Native != nil {
		count++
	}
	return count
}

// Structs to handle parsed bidder info, so we aren't reparsing every request
type parsedBidderInfo struct {
	app         parsedSupports
	site        parsedSupports
	dooh        parsedSupports
	multiformat bool
}

type parsedSupports struct {
	enabled bool
	banner  bool
	video   bool
	audio   bool
	native  bool
}

func parseBidderInfo(info config.BidderInfo) parsedBidderInfo {
	var parsedInfo parsedBidderInfo

	if info.Capabilities == nil {
		return parsedInfo
	}

	if info.Capabilities.App != nil {
		parsedInfo.app.enabled = true
		parsedInfo.app.banner, parsedInfo.app.video, parsedInfo.app.audio, parsedInfo.app.native = parseAllowedTypes(info.Capabilities.App.MediaTypes)
	}
	if info.Capabilities.Site != nil {
		parsedInfo.site.enabled = true
		parsedInfo.site.banner, parsedInfo.site.video, parsedInfo.site.audio, parsedInfo.site.native = parseAllowedTypes(info.Capabilities.Site.MediaTypes)
	}
	if info.Capabilities.DOOH != nil {
		parsedInfo.dooh.enabled = true
		parsedInfo.dooh.banner, parsedInfo.dooh.video, parsedInfo.dooh.audio, parsedInfo.dooh.native = parseAllowedTypes(info.Capabilities.DOOH.MediaTypes)
	}
	parsedInfo.multiformat = IsMultiFormatSupported(info)

	return parsedInfo
}

// adjustImpForPreferredMediaType modifies the given impression to retain only the preferred media type.
// It returns the updated impression and any error encountered during the adjustment process.
func adjustImpForPreferredMediaType(imp *openrtb2.Imp, preferredMediaType openrtb_ext.BidType) (bool, []error) {
	var errors []error
	var removeImp bool
	// Clear irrelevant media types based on the preferred media type.
	switch preferredMediaType {
	case openrtb_ext.BidTypeBanner:
		if imp.Banner != nil {
			removeVideo(imp, &errors)
			removeAudio(imp, &errors)
			removeNative(imp, &errors)
		} else {
			return true, []error{&errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a preferred BANNER media type. It will be ignored.", imp.ID)}}
		}
	case openrtb_ext.BidTypeVideo:
		if imp.Video != nil {
			removeBanner(imp, &errors)
			removeAudio(imp, &errors)
			removeNative(imp, &errors)
		} else {
			return true, []error{&errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a preferred VIDEO media type. It will be ignored.", imp.ID)}}
		}
	case openrtb_ext.BidTypeAudio:
		if imp.Audio != nil {
			removeBanner(imp, &errors)
			removeVideo(imp, &errors)
			removeNative(imp, &errors)
		} else {
			return true, []error{&errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a preferred AUDIO media type. It will be ignored.", imp.ID)}}
		}
	case openrtb_ext.BidTypeNative:
		if imp.Native != nil {
			removeBanner(imp, &errors)
			removeVideo(imp, &errors)
			removeAudio(imp, &errors)
		} else {
			return true, []error{&errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a preferred NATIVE media type. It will be ignored.", imp.ID)}}
		}
	case "":
		return true, []error{&errortypes.BadInput{Message: fmt.Sprintf("Removing the imp %s as the bidder does not support multi-format and preferred media type is not defined for the bidder", imp.ID)}}
	default:
		return true, []error{&errortypes.BadInput{Message: fmt.Sprintf("Imp %s has an invalid preferred media type: %s. It will be ignored.", imp.ID, preferredMediaType)}}
	}

	return removeImp, errors
}

// Function to remove the banner media type from the impression and add a warning.
func removeBanner(imp *openrtb2.Imp, errors *[]error) {
	if imp.Banner != nil {
		addWarning(errors, imp.ID, openrtb_ext.BidTypeBanner)
		imp.Banner = nil
	}
}

// Function to remove the video media type from the impression and add a warning.
func removeVideo(imp *openrtb2.Imp, errors *[]error) {
	if imp.Video != nil {
		addWarning(errors, imp.ID, openrtb_ext.BidTypeVideo)
		imp.Video = nil
	}
}

// Function to remove the Native media type from the impression and add a warning.
func removeNative(imp *openrtb2.Imp, errors *[]error) {
	if imp.Native != nil {
		addWarning(errors, imp.ID, openrtb_ext.BidTypeNative)
		imp.Native = nil
	}
}

// Function to remove the Audio media type from the impression and add a warning.
func removeAudio(imp *openrtb2.Imp, errors *[]error) {
	if imp.Audio != nil {
		addWarning(errors, imp.ID, openrtb_ext.BidTypeAudio)
		imp.Audio = nil
	}
}

// Function to add a warning message to the list.
func addWarning(errors *[]error, impID string, mediaTypeName openrtb_ext.BidType) {
	*errors = append(*errors, &errortypes.Warning{Message: fmt.Sprintf("Imp %s uses %s, removing %s as the bidder doesn't support multi-format", impID, mediaTypeName, mediaTypeName)})
}

func IsMultiFormatSupported(bidderInfo config.BidderInfo) bool {
	if bidderInfo.OpenRTB != nil && bidderInfo.OpenRTB.MultiformatSupported != nil {
		return *bidderInfo.OpenRTB.MultiformatSupported
	}
	return true
}
