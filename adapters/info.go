package adapters

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	yaml "gopkg.in/yaml.v2"
)

// EnforceBidderInfo decorates the input Bidder by making sure that all the requests
// to it are in sync with its static/bidder-info/{bidder}.yaml file.
//
// It adjusts incoming requests in the following ways:
//   1. If App or Site traffic is not supported by the info file, then requests from
//      those sources will be rejected before the delegate is called.
//   2. If a given MediaType is not supported for the platform, then it will be set
//      to nil before the request is forwarded to the delegate.
//   3. Any Imps which have no MediaTypes left will be removed.
//   4. If there are no valid Imps left, the delegate won't be called at all.
func EnforceBidderInfo(bidder Bidder, info BidderInfo) Bidder {
	return &InfoAwareBidder{
		Bidder: bidder,
		info:   info,
	}
}

type InfoAwareBidder struct {
	Bidder
	info BidderInfo
}

func (i *InfoAwareBidder) MakeRequests(request *openrtb.BidRequest) ([]*RequestData, []error) {
	var allowedMediaTypes []openrtb_ext.BidType
	if request.Site != nil {
		if i.info.Capabilities.Site == nil {
			return nil, []error{BadInput("this bidder does not support site requests")}
		}
		allowedMediaTypes = i.info.Capabilities.Site.MediaTypes
	}
	if request.App != nil {
		if i.info.Capabilities.App == nil {
			return nil, []error{BadInput("this bidder does not support app requests")}
		}
		allowedMediaTypes = i.info.Capabilities.App.MediaTypes
	}

	// Filtering imps is quite expensive (array filter with large, non-pointer elements)... but should be rare,
	// because it only happens if the publisher makes a really bad request.
	//
	// To avoid allocating new arrays and copying in the normal case, we'll make one pass to
	// see if any imps need to be removed, and another to do the removing if necessary.
	numToFilter, errs := i.pruneImps(request.Imp, allowedMediaTypes)
	if numToFilter != 0 {
		filteredImps, newErrs := i.filterImps(request.Imp, numToFilter)
		request.Imp = filteredImps
		errs = append(errs, newErrs...)
	}
	reqs, delegateErrs := i.Bidder.MakeRequests(request)
	return reqs, append(errs, delegateErrs...)
}

// pruneImps trims invalid media types from each imp, and returns true if any of the
// Imps have _no_ valid Media Types left.
func (i *InfoAwareBidder) pruneImps(imps []openrtb.Imp, allowedTypes []openrtb_ext.BidType) (int, []error) {
	allowBanner, allowVideo, allowAudio, allowNative := parseAllowedTypes(allowedTypes)
	numToFilter := 0
	var errs []error
	for i := 0; i < len(imps); i++ {
		if !allowBanner && imps[i].Banner != nil {
			imps[i].Banner = nil
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] uses banner, but this bidder doesn't support it", i)))
		}
		if !allowVideo && imps[i].Video != nil {
			imps[i].Video = nil
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] uses video, but this bidder doesn't support it", i)))
		}
		if !allowAudio && imps[i].Audio != nil {
			imps[i].Audio = nil
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] uses audio, but this bidder doesn't support it", i)))
		}
		if !allowNative && imps[i].Native != nil {
			imps[i].Native = nil
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] uses native, but this bidder doesn't support it", i)))
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

func hasAnyTypes(imp *openrtb.Imp) bool {
	return imp.Banner != nil || imp.Video != nil || imp.Audio != nil || imp.Native != nil
}

func (i *InfoAwareBidder) filterImps(imps []openrtb.Imp, numToFilter int) ([]openrtb.Imp, []error) {
	newImps := make([]openrtb.Imp, 0, len(imps)-numToFilter)
	errs := make([]error, 0, numToFilter)
	for i := 0; i < len(imps); i++ {
		if hasAnyTypes(&imps[i]) {
			newImps = append(newImps, imps[i])
		} else {
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] has no supported MediaTypes. It will be ignored", i)))
		}
	}
	return newImps, errs
}

type BidderInfos map[string]BidderInfo

// ParseBidderInfos reads all the static/bidder-info/{bidder}.yaml files from the filesystem.
// The map it returns will have a key for every element of the bidders array.
// If a {bidder}.yaml file does not exist for some bidder, it will panic.
func ParseBidderInfos(infoDir string, bidders []openrtb_ext.BidderName) BidderInfos {
	bidderInfos := make(map[string]BidderInfo, len(bidders))
	for _, bidderName := range bidders {
		bidderString := string(bidderName)
		fileData, err := ioutil.ReadFile(infoDir + "/" + bidderString + ".yaml")
		if err != nil {
			glog.Fatalf("error reading from file %s: %v", infoDir+"/"+bidderString+".yaml", err)
		}

		var parsedInfo BidderInfo
		if err := yaml.Unmarshal(fileData, &parsedInfo); err != nil {
			glog.Fatalf("error parsing yaml in file %s: %v", infoDir+"/"+bidderString+".yaml", err)
		}
		bidderInfos[bidderString] = parsedInfo
	}
	return bidderInfos
}

func (infos BidderInfos) HasAppSupport(bidder openrtb_ext.BidderName) bool {
	return infos[string(bidder)].Capabilities.App != nil
}

func (infos BidderInfos) HasSiteSupport(bidder openrtb_ext.BidderName) bool {
	return infos[string(bidder)].Capabilities.Site != nil
}

func (infos BidderInfos) SupportsAppMediaType(bidder openrtb_ext.BidderName, mediaType openrtb_ext.BidType) bool {
	return containsMediaType(infos[string(bidder)].Capabilities.App.MediaTypes, mediaType)
}

func (infos BidderInfos) SupportsWebMediaType(bidder openrtb_ext.BidderName, mediaType openrtb_ext.BidType) bool {
	return containsMediaType(infos[string(bidder)].Capabilities.Site.MediaTypes, mediaType)
}

type BidderInfo struct {
	Maintainer   *MaintainerInfo   `yaml:"maintainer" json:"maintainer"`
	Capabilities *CapabilitiesInfo `yaml:"capabilities" json:"capabilities"`
}

type MaintainerInfo struct {
	Email string `yaml:"email" json:"email"`
}

type CapabilitiesInfo struct {
	App  *PlatformInfo `yaml:"app" json:"app"`
	Site *PlatformInfo `yaml:"site" json:"site"`
}

type PlatformInfo struct {
	MediaTypes []openrtb_ext.BidType `yaml:"mediaTypes" json:"mediaTypes"`
}

func containsMediaType(haystack []openrtb_ext.BidType, needle openrtb_ext.BidType) bool {
	for i := 0; i < len(haystack); i++ {
		if needle == haystack[i] {
			return true
		}
	}
	return false
}
