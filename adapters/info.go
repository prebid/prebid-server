package adapters

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
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
		info:   parseBidderInfo(info),
	}
}

type InfoAwareBidder struct {
	Bidder
	info parsedBidderInfo
}

func (i *InfoAwareBidder) MakeRequests(request *openrtb.BidRequest, reqInfo *ExtraRequestInfo) ([]*RequestData, []error) {
	var allowedMediaTypes parsedSupports

	if request.Site != nil {
		if !i.info.site.enabled {
			return nil, []error{BadInput("this bidder does not support site requests")}
		}
		allowedMediaTypes = i.info.site
	}
	if request.App != nil {
		if !i.info.app.enabled {
			return nil, []error{BadInput("this bidder does not support app requests")}
		}
		allowedMediaTypes = i.info.app
	}

	// Filtering imps is quite expensive (array filter with large, non-pointer elements)... but should be rare,
	// because it only happens if the publisher makes a really bad request.
	//
	// To avoid allocating new arrays and copying in the normal case, we'll make one pass to
	// see if any imps need to be removed, and another to do the removing if necessary.
	numToFilter, errs := i.pruneImps(request.Imp, allowedMediaTypes)

	// If all imps in bid request come with unsupported media types, exit
	if numToFilter == len(request.Imp) {
		return nil, append(errs, BadInput("Bid request didn't contain media types supported by the bidder"))
	}

	if numToFilter != 0 {
		// Filter out imps with unsupported media types
		filteredImps, newErrs := i.filterImps(request.Imp, numToFilter)
		request.Imp = filteredImps
		errs = append(errs, newErrs...)
	}
	reqs, delegateErrs := i.Bidder.MakeRequests(request, reqInfo)
	return reqs, append(errs, delegateErrs...)
}

// pruneImps trims invalid media types from each imp, and returns true if any of the
// Imps have _no_ valid Media Types left.
func (i *InfoAwareBidder) pruneImps(imps []openrtb.Imp, allowedTypes parsedSupports) (int, []error) {
	numToFilter := 0
	var errs []error
	for i := 0; i < len(imps); i++ {
		if !allowedTypes.banner && imps[i].Banner != nil {
			imps[i].Banner = nil
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] uses banner, but this bidder doesn't support it", i)))
		}
		if !allowedTypes.video && imps[i].Video != nil {
			imps[i].Video = nil
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] uses video, but this bidder doesn't support it", i)))
		}
		if !allowedTypes.audio && imps[i].Audio != nil {
			imps[i].Audio = nil
			errs = append(errs, BadInput(fmt.Sprintf("request.imp[%d] uses audio, but this bidder doesn't support it", i)))
		}
		if !allowedTypes.native && imps[i].Native != nil {
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
func ParseBidderInfos(cfg map[string]config.Adapter, infoDir string, bidders []openrtb_ext.BidderName) BidderInfos {
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

		if isEnabledBidder(cfg, bidderString) {
			parsedInfo.Status = StatusActive
		} else {
			parsedInfo.Status = StatusDisabled
		}

		bidderInfos[bidderString] = parsedInfo
	}
	return bidderInfos
}

func (infos BidderInfos) IsActive(bidder openrtb_ext.BidderName) bool {
	return infos[string(bidder)].Status == StatusActive
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

// isEnabledBidder Checks that a bidder config exists and is not disabled
func isEnabledBidder(cfg map[string]config.Adapter, bidder string) bool {
	a, ok := cfg[strings.ToLower(bidder)]
	return ok && !a.Disabled
}

// BidderStatus represents a bidder status in PBS, can be either active or disabled
type BidderStatus string

const (
	StatusUnknown  BidderStatus = ""
	StatusActive   BidderStatus = "ACTIVE"
	StatusDisabled BidderStatus = "DISABLED"
)

type BidderInfo struct {
	Status                  BidderStatus      `yaml:"status" json:"status"`
	Maintainer              *MaintainerInfo   `yaml:"maintainer" json:"maintainer"`
	Capabilities            *CapabilitiesInfo `yaml:"capabilities" json:"capabilities"`
	AliasOf                 string            `json:"aliasOf,omitempty"`
	ModifyingVastXmlAllowed bool              `yaml:"modifyingVastXmlAllowed" json:"-" xml:"-"`
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

func parseBidderInfo(info BidderInfo) parsedBidderInfo {
	var parsedInfo parsedBidderInfo
	if info.Capabilities.App != nil {
		parsedInfo.app.enabled = true
		parsedInfo.app.banner, parsedInfo.app.video, parsedInfo.app.audio, parsedInfo.app.native = parseAllowedTypes(info.Capabilities.App.MediaTypes)
	}
	if info.Capabilities.Site != nil {
		parsedInfo.site.enabled = true
		parsedInfo.site.banner, parsedInfo.site.video, parsedInfo.site.audio, parsedInfo.site.native = parseAllowedTypes(info.Capabilities.Site.MediaTypes)
	}
	return parsedInfo
}

type ExtraRequestInfo struct {
	PbsEntryPoint pbsmetrics.RequestType
}
