package adapters

import (
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/openrtb_ext"
	yaml "gopkg.in/yaml.v2"
)

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
