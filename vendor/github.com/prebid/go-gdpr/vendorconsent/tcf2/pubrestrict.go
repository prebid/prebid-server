package vendorconsent

import (
	"fmt"

	"github.com/prebid/go-gdpr/bitutils"
)

// IAB spec does not specify a max vendorID for the publisher restrictions. This should be one bit short of the max possible.
const assumedMaxVendorID uint16 = 32767

func parsePubRestriction(metadata ConsentMetadata, startbit uint) (*pubRestrictions, uint, error) {
	data := metadata.data
	numRestrictions, err := bitutils.ParseUInt12(data, startbit)
	if err != nil {
		return nil, 0, fmt.Errorf("Error on parsing the number of publisher restrictions: %s", err.Error())
	}

	// Parse out the "exceptions" here.
	currentOffset := startbit + 12
	restrictions := make(map[byte]pubRestriction, numRestrictions)
	for j := uint16(0); j < numRestrictions; j++ {
		restrictData, err := bitutils.ParseByte8(data, currentOffset)
		if err != nil {
			return nil, 0, fmt.Errorf("Error on parsing the publisher restriction purpose/type: %s", err.Error())
		}
		currentOffset = currentOffset + 8
		numEntries, err := bitutils.ParseUInt12(data, currentOffset)
		if err != nil {
			return nil, 0, fmt.Errorf("Error on parsing the number of publisher restriction vendor ranges: %s", err.Error())
		}
		currentOffset = currentOffset + 12
		vendors := make([]rangeConsent, numEntries)
		for i := uint16(0); i < numEntries; i++ {
			thisConsent, bitsConsumed, err := parseRangeConsent(data, currentOffset, assumedMaxVendorID)
			if err != nil {
				return nil, 0, err
			}
			vendors[i] = thisConsent
			currentOffset = currentOffset + bitsConsumed
		}
		restrictions[restrictData] = pubRestriction{
			purposeID:    (restrictData & 0xfc) >> 2,
			restrictType: (restrictData & 0x03),
			vendors:      vendors,
		}
	}
	return &pubRestrictions{restrictions: restrictions}, currentOffset, nil
}

type pubRestrictions struct {
	restrictions map[byte]pubRestriction
}

type pubRestriction struct {
	purposeID    uint8
	restrictType uint8
	vendors      []rangeConsent
}

func (p *pubRestrictions) CheckPubRestriction(purposeID uint8, restrictType uint8, vendor uint16) bool {
	key := byte(purposeID<<2 | (restrictType & 0x03))
	restriction, ok := p.restrictions[key]
	if !ok {
		return false
	}
	for i := 0; i < len(restriction.vendors); i++ {
		if restriction.vendors[i].Contains(vendor) {
			return true
		}
	}
	return false

}
