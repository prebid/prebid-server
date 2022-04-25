package vendorconsent

import (
	"fmt"

	"github.com/prebid/go-gdpr/bitutils"
)

func parseRangeSection(metadata ConsentMetadata, maxVendorID uint16, startbit uint) (*rangeSection, uint, error) {
	data := metadata.data

	if len(data) < 31 {
		return nil, 0, fmt.Errorf("vendor consent strings using RangeSections require at least 31 bytes. Got %d", len(data))
	}

	// This makes an int from bits [startBit, startBit + 12)
	numEntries, err := bitutils.ParseUInt12(data, startbit)
	if err != nil {
		return nil, 0, err
	}

	// Parse out the "exceptions" here.
	currentOffset := startbit + 12
	consents := make([]rangeConsent, numEntries)
	for i := range consents {
		bitsConsumed, err := parseRangeConsent(&consents[i], data, currentOffset, maxVendorID)
		if err != nil {
			return nil, 0, err
		}
		currentOffset = currentOffset + bitsConsumed
	}

	return &rangeSection{
		consents:    consents,
		maxVendorID: maxVendorID,
	}, currentOffset, nil
}

// RangeSection Exception implementations

// parseRangeConsents parses a RangeSection starting from the initial bit.
// It returns the number of bits consumed by the parsing.
func parseRangeConsent(dst *rangeConsent, data []byte, initialBit uint, maxVendorID uint16) (uint, error) {
	// Fixes #10
	if uint(len(data)) <= initialBit/8 {
		return 0, fmt.Errorf("bit %d was supposed to start a new RangeEntry, but the consent string was only %d bytes long", initialBit, len(data))
	}
	// If the first bit is set, it's a Range of IDs
	if isSet(data, initialBit) {
		start, err := bitutils.ParseUInt16(data, initialBit+1)
		if err != nil {
			return 0, err
		}
		end, err := bitutils.ParseUInt16(data, initialBit+17)
		if err != nil {
			return 0, err
		}
		if start == 0 {
			return 0, fmt.Errorf("bit %d range entry exclusion starts at 0, but the min vendor ID is 1", initialBit)
		}
		if end > maxVendorID {
			return 0, fmt.Errorf("bit %d range entry exclusion ends at %d, but the max vendor ID is %d", initialBit, end, maxVendorID)
		}
		if end <= start {
			return 0, fmt.Errorf("bit %d range entry excludes vendors [%d, %d]. The start should be less than the end", initialBit, start, end)
		}
		dst.startID = start
		dst.endID = end
		return 33, nil
	}

	vendorID, err := bitutils.ParseUInt16(data, initialBit+1)
	if err != nil {
		return 0, err
	}
	if vendorID == 0 || vendorID > maxVendorID {
		return 0, fmt.Errorf("bit %d range entry excludes vendor %d, but only vendors [1, %d] are valid", initialBit, vendorID, maxVendorID)
	}

	dst.startID = vendorID
	dst.endID = vendorID
	return 17, nil
}

// A RangeConsents encodes consents that have been registered.
type rangeSection struct {
	consents    []rangeConsent
	maxVendorID uint16
}

func (p *rangeSection) MaxVendorID() uint16 {
	if p == nil {
		return 0
	}

	return p.maxVendorID
}

// VendorConsents implementation
func (p *rangeSection) VendorConsent(id uint16) bool {
	if id < 1 || id > p.maxVendorID {
		return false
	}

	for i := range p.consents {
		if p.consents[i].Contains(id) {
			return true
		}
	}
	return false
}

// This is a RangeSection exception for a range of IDs.
// The start and end bounds here are inclusive.
type rangeConsent struct {
	startID uint16
	endID   uint16
}

func (e rangeConsent) Contains(id uint16) bool {
	return e.startID <= id && e.endID >= id
}
