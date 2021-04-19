package adapters

import (
	"fmt"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// InfoAwareBidder wraps a Bidder to ensure all requests abide by the capabilities and
// media types defined in the static/bidder-info/{bidder}.yaml file.
//
// It adjusts incoming requests in the following ways:
//   1. If App or Site traffic is not supported by the info file, then requests from
//      those sources will be rejected before the delegate is called.
//   2. If a given MediaType is not supported for the platform, then it will be set
//      to nil before the request is forwarded to the delegate.
//   3. Any Imps which have no MediaTypes left will be removed.
//   4. If there are no valid Imps left, the delegate won't be called at all.
type InfoAwareBidder struct {
	Bidder
	info parsedBidderInfo
}

// BuildInfoAwareBidder wraps a bidder to enforce site, app, and media type support.
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
			return nil, []error{&errortypes.BadInput{Message: "this bidder does not support site requests"}}
		}
		allowedMediaTypes = i.info.site
	}
	if request.App != nil {
		if !i.info.app.enabled {
			return nil, []error{&errortypes.BadInput{Message: "this bidder does not support app requests"}}
		}
		allowedMediaTypes = i.info.app
	}

	// Filtering imps is quite expensive (array filter with large, non-pointer elements)... but should be rare,
	// because it only happens if the publisher makes a really bad request.
	//
	// To avoid allocating new arrays and copying in the normal case, we'll make one pass to
	// see if any imps need to be removed, and another to do the removing if necessary.
	numToFilter, errs := pruneImps(request.Imp, allowedMediaTypes)

	// If all imps in bid request come with unsupported media types, exit
	if numToFilter == len(request.Imp) {
		return nil, append(errs, &errortypes.BadInput{Message: "Bid request didn't contain media types supported by the bidder"})
	}

	if numToFilter != 0 {
		// Filter out imps with unsupported media types
		filteredImps, newErrs := filterImps(request.Imp, numToFilter)
		request.Imp = filteredImps
		errs = append(errs, newErrs...)
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
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d] uses banner, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.video && imps[i].Video != nil {
			imps[i].Video = nil
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d] uses video, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.audio && imps[i].Audio != nil {
			imps[i].Audio = nil
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d] uses audio, but this bidder doesn't support it", i)})
		}
		if !allowedTypes.native && imps[i].Native != nil {
			imps[i].Native = nil
			errs = append(errs, &errortypes.BadInput{Message: fmt.Sprintf("request.imp[%d] uses native, but this bidder doesn't support it", i)})
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
	app  parsedSupports
	site parsedSupports
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
	if info.Capabilities != nil && info.Capabilities.App != nil {
		parsedInfo.app.enabled = true
		parsedInfo.app.banner, parsedInfo.app.video, parsedInfo.app.audio, parsedInfo.app.native = parseAllowedTypes(info.Capabilities.App.MediaTypes)
	}
	if info.Capabilities != nil && info.Capabilities.Site != nil {
		parsedInfo.site.enabled = true
		parsedInfo.site.banner, parsedInfo.site.video, parsedInfo.site.audio, parsedInfo.site.native = parseAllowedTypes(info.Capabilities.Site.MediaTypes)
	}
	return parsedInfo
}
