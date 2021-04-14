package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/prebid/prebid-server/openrtb_ext"
	"gopkg.in/yaml.v2"
)

// BidderInfos contains a mapping of bidder name to bidder info.
type BidderInfos map[string]BidderInfo

// BidderInfo is the maintainer information, supported auction types, and feature opts-in for a bidder.
type BidderInfo struct {
	Enabled                 bool              // copied from adapter config for convenience. to be refactored.
	Maintainer              *MaintainerInfo   `yaml:"maintainer"`
	Capabilities            *CapabilitiesInfo `yaml:"capabilities"`
	ModifyingVastXmlAllowed bool              `yaml:"modifyingVastXmlAllowed"`
	Debug                   *DebugInfo        `yaml:"debug,omitempty"`
	GVLVendorID             uint16            `yaml:"gvlVendorID,omitempty"`
}

// MaintainerInfo is the support email address for a bidder.
type MaintainerInfo struct {
	Email string `yaml:"email"`
}

// CapabilitiesInfo is the supported platforms for a bidder.
type CapabilitiesInfo struct {
	App  *PlatformInfo `yaml:"app"`
	Site *PlatformInfo `yaml:"site"`
}

// PlatformInfo is the supported media types for a bidder.
type PlatformInfo struct {
	MediaTypes []openrtb_ext.BidType `yaml:"mediaTypes"`
}

// DebugInfo is the supported debug options for a bidder.
type DebugInfo struct {
	Allow bool `yaml:"allow"`
}

// LoadBidderInfoFromDisk parses all static/bidder-info/{bidder}.yaml files from the file system.
func LoadBidderInfoFromDisk(path string, adapterConfigs map[string]Adapter, bidders []string) (BidderInfos, error) {
	reader := infoReaderFromDisk{path}
	return loadBidderInfo(reader, adapterConfigs, bidders)
}

func loadBidderInfo(r infoReader, adapterConfigs map[string]Adapter, bidders []string) (BidderInfos, error) {
	infos := BidderInfos{}

	for _, bidder := range bidders {
		data, err := r.Read(bidder)
		if err != nil {
			return nil, err
		}

		info := BidderInfo{}
		if err := yaml.Unmarshal(data, &info); err != nil {
			return nil, fmt.Errorf("error parsing yaml for bidder %s: %v", bidder, err)
		}

		info.Enabled = isEnabledByConfig(adapterConfigs, bidder)
		infos[bidder] = info
	}

	return infos, nil
}

func isEnabledByConfig(adapterConfigs map[string]Adapter, bidderName string) bool {
	a, ok := adapterConfigs[strings.ToLower(bidderName)]
	return ok && !a.Disabled
}

type infoReader interface {
	Read(bidder string) ([]byte, error)
}

type infoReaderFromDisk struct {
	path string
}

func (r infoReaderFromDisk) Read(bidder string) ([]byte, error) {
	path := fmt.Sprintf("%v/%v.yaml", r.path, bidder)
	return ioutil.ReadFile(path)
}

// ToGVLVendorIDMap transforms a BidderInfos object to a map of bidder names to GVL id. Disabled
// bidders are omitted from the result.
func (infos BidderInfos) ToGVLVendorIDMap() map[openrtb_ext.BidderName]uint16 {
	m := make(map[openrtb_ext.BidderName]uint16, len(infos))
	for name, info := range infos {
		if info.Enabled && info.GVLVendorID != 0 {
			m[openrtb_ext.BidderName(name)] = info.GVLVendorID
		}
	}
	return m
}
