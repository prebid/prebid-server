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

	// Filtering imps is quite expensive (array filter with large, non-pointer elements)... but should be rare,
	// because it only happens if the publisher makes a really bad request.
	//
	// To avoid allocating new arrays and copying in the normal case, we'll make one pass to
	// see if any imps need to be removed, and another to do the removing if necessary.
	numToFilter, errs := pruneImps(request.Imp, allowedMediaTypes)

	// If all imps in bid request come with unsupported media types, exit
	if numToFilter == len(request.Imp) {
		return nil, append(errs, &errortypes.Warning{Message: "Bid request didn't contain media types supported by the bidder"})
	}

	if numToFilter != 0 {
		// Filter out imps with unsupported media types
		filteredImps, newErrs := filterImps(request.Imp, numToFilter)
		request.Imp = filteredImps
		errs = append(errs, newErrs...)
	}

	//if bidder doesnt support multiformat, send only preferred media type in the request
	if !i.info.multiformat {
		var newErrs []error
		request.Imp, newErrs = FilterMultiformatImps(request, reqInfo.PreferredMediaType)
		if newErrs != nil {
			errs = append(errs, newErrs...)
		}
	}

	reqs, delegateErrs := i.Bidder.MakeRequests(request, reqInfo)
	return reqs, append(errs, delegateErrs...)
}

// pruneImps trims invalid media types from each imp, and returns true if any of the
// Imps have _no_ valid Media Types left.
func pruneImps(imps []openrtb2.Imp, allowedTypes parsedSupports) (int, []error) {
	numToFilter := 0
	var errs []error
	for i := 0; i < len(imps); i++ {
		if !allowedTypes.banner && imps[i].Banner != nil {
			imps[i].Banner = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses banner, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.video && imps[i].Video != nil {
			imps[i].Video = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses video, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.audio && imps[i].Audio != nil {
			imps[i].Audio = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses audio, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.native && imps[i].Native != nil {
			imps[i].Native = nil
			errs = append(errs, &errortypes.Warning{Message: fmt.Sprintf("request.imp[%d] uses native, but this bidder doesn't support it", i)})
		}
		if !hasAnyTypes(&imps[i]) {
			numToFilter = numToFilter + 1
		}
	}
	return numToFilter, errs
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

func hasAnyTypes(imp *openrtb2.Imp) bool {
	return imp.Banner != nil || imp.Video != nil || imp.Audio != nil || imp.Native != nil
}

func filterImps(imps []openrtb2.Imp, numToFilter int) ([]openrtb2.Imp, []error) {
	newImps := make([]openrtb2.Imp, 0, len(imps)-numToFilter)
	errs := make([]error, 0, numToFilter)
	for i := 0; i < len(imps); i++ {
		if hasAnyTypes(&imps[i]) {
			newImps = append(newImps, imps[i])
		} else {
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d] has no supported MediaTypes. It will be ignored", i)})
		}
	}
	return newImps, errs
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

// filterMultiformatSupported should read bidder info for multi-format support and if bidder does not support multi-format requests, then send only prefered media type in the request
func FilterMultiformatImps(bidRequest *openrtb2.BidRequest, preferredMediaType openrtb_ext.BidType) ([]openrtb2.Imp, []error) {

	var updatedImps []openrtb2.Imp
	var errs []error

	for _, imp := range bidRequest.Imp {
		processedImp, err := AdjustImpForPreferredMediaType(imp, preferredMediaType)
		if err != nil {
			errs = append(errs, err)
		} else {
			updatedImps = append(updatedImps, *processedImp)
		}
	}

	if len(updatedImps) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "Bid request contains 0 impressions after filtering."})
	}

	return updatedImps, errs
}

func AdjustImpForPreferredMediaType(imp openrtb2.Imp, preferredMediaType openrtb_ext.BidType) (*openrtb2.Imp, error) {
	if !IsMultiFormat(imp) {
		// If the impression is not multi-format, return it as-is.
		return &imp, nil
	}

	if preferredMediaType == "" {
		return &imp, nil
	}

	// Create a copy of the Imp and clear irrelevant media types based on the preferred media type.
	updatedImp := imp

	switch preferredMediaType {
	case openrtb_ext.BidTypeBanner:
		if imp.Banner != nil {
			updatedImp.Video = nil
			updatedImp.Audio = nil
			updatedImp.Native = nil
		} else {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a valid BANNER media type.", imp.ID)}
		}
	case openrtb_ext.BidTypeVideo:
		if imp.Video != nil {
			updatedImp.Banner = nil
			updatedImp.Audio = nil
			updatedImp.Native = nil
		} else {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a valid VIDEO media type.", imp.ID)}
		}
	case openrtb_ext.BidTypeAudio:
		if imp.Audio != nil {
			updatedImp.Banner = nil
			updatedImp.Video = nil
			updatedImp.Native = nil
		} else {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a valid AUDIO media type.", imp.ID)}
		}
	case openrtb_ext.BidTypeNative:
		if imp.Native != nil {
			updatedImp.Banner = nil
			updatedImp.Video = nil
			updatedImp.Audio = nil
		} else {
			return nil, &errortypes.BadInput{Message: fmt.Sprintf("Imp %s does not have a valid NATIVE media type.", imp.ID)}
		}
	default:
		return nil, &errortypes.BadInput{Message: fmt.Sprintf("Imp %s has an invalid preferred media type: %s.", imp.ID, preferredMediaType)}
	}

	return &updatedImp, nil
}

func IsMultiFormatSupported(bidderInfo config.BidderInfo) bool {
	if bidderInfo.OpenRTB != nil {
		return bidderInfo.OpenRTB.MultiformatSupported
	}
	return true
}

func IsMultiFormat(imp openrtb2.Imp) bool {
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
	return count > 1
}
