package openrtb_ext

import (
	"fmt"
	"strings"
)

// ExtAlternateBidderCodes defines list of alternate bidder codes allowed by adatpers. This overrides host level configs.
type ExtAlternateBidderCodes struct {
	Enabled bool                                      `mapstructure:"enabled" json:"enabled"`
	Bidders map[string]ExtAdapterAlternateBidderCodes `mapstructure:"bidders" json:"bidders"`
}

type ExtAdapterAlternateBidderCodes struct {
	Enabled            bool     `mapstructure:"enabled" json:"enabled"`
	AllowedBidderCodes []string `mapstructure:"allowedbiddercodes" json:"allowedbiddercodes"`
}

func (bidderCodes *ExtAlternateBidderCodes) IsValidBidderCode(bidder, alternateBidder string) (bool, error) {
	if alternateBidder == "" || strings.EqualFold(bidder, alternateBidder) {
		return true, nil
	}

	if !bidderCodes.Enabled {
		return false, alternateBidderDisabledError(bidder, alternateBidder)
	}

	if bidderCodes.Bidders == nil {
		return false, alternateBidderNotDefinedError(bidder, alternateBidder)
	}

	adapterCfg, found := bidderCodes.IsBidderInAlternateBidderCodes(bidder)
	if !found {
		return false, alternateBidderNotDefinedError(bidder, alternateBidder)
	}

	if !adapterCfg.Enabled {
		// config has bidder entry but is not enabled, report it
		return false, alternateBidderDisabledError(bidder, alternateBidder)
	}

	if adapterCfg.AllowedBidderCodes == nil || (len(adapterCfg.AllowedBidderCodes) == 1 && adapterCfg.AllowedBidderCodes[0] == "*") {
		return true, nil
	}

	for _, code := range adapterCfg.AllowedBidderCodes {
		if alternateBidder == code {
			return true, nil
		}
	}

	return false, fmt.Errorf("invalid biddercode %q sent by adapter %q", alternateBidder, bidder)
}

func alternateBidderDisabledError(bidder, alternateBidder string) error {
	return fmt.Errorf("alternateBidderCodes disabled for %q, rejecting bids for %q", bidder, alternateBidder)
}

func alternateBidderNotDefinedError(bidder, alternateBidder string) error {
	return fmt.Errorf("alternateBidderCodes not defined for adapter %q, rejecting bids for %q", bidder, alternateBidder)
}

// IsBidderInAlternateBidderCodes tries to find bidder in the altBidderCodes.Bidders map in a case sensitive
// manner first. If no match is found it'll try it in a case insensitive way in linear time
func (bidderCodes *ExtAlternateBidderCodes) IsBidderInAlternateBidderCodes(bidder string) (ExtAdapterAlternateBidderCodes, bool) {
	if len(bidder) > 0 && bidderCodes != nil && len(bidderCodes.Bidders) > 0 {
		// try constant time exact match
		if adapterCfg, found := bidderCodes.Bidders[bidder]; found {
			return adapterCfg, true
		}

		// check if we can find with a case insensitive comparison
		for bidderName, adapterCfg := range bidderCodes.Bidders {
			if strings.EqualFold(bidder, bidderName) {
				return adapterCfg, true
			}
		}
	}

	return ExtAdapterAlternateBidderCodes{}, false
}
