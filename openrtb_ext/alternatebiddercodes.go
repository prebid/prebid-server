package openrtb_ext

import "fmt"

// ExtAlternateBidderCodes defines list of alternate bidder codes allowed by adatpers. This overrides host level configs.
type ExtAlternateBidderCodes struct {
	Enabled bool                                      `mapstructure:"enabled" json:"enabled"`
	Bidders map[string]ExtAdapterAlternateBidderCodes `mapstructure:"bidders" json:"bidders"`
}

type ExtAdapterAlternateBidderCodes struct {
	Enabled            bool     `mapstructure:"enabled" json:"enabled"`
	AllowedBidderCodes []string `mapstructure:"allowedbiddercodes" json:"allowedbiddercodes"`
}

func IsValidBidderCode(bidderCodes *ExtAlternateBidderCodes, bidder, alternateBidder string) (bool, error) {
	const ErrAlternateBidderNotDefined = "alternateBidderCodes not defined for adapter %q, rejecting bids for %q"
	const ErrAlternateBidderDisabled = "alternateBidderCodes disabled for %q, rejecting bids for %q"

	if alternateBidder == "" || bidder == alternateBidder {
		return true, nil
	}

	if bidderCodes == nil {
		return false, fmt.Errorf(ErrAlternateBidderNotDefined, bidder, alternateBidder)
	}

	if !bidderCodes.Enabled {
		return false, fmt.Errorf(ErrAlternateBidderDisabled, bidder, alternateBidder)
	}

	if bidderCodes.Bidders == nil {
		return false, fmt.Errorf(ErrAlternateBidderNotDefined, bidder, alternateBidder)
	}

	adapterCfg, ok := bidderCodes.Bidders[bidder]
	if !ok {
		return false, fmt.Errorf(ErrAlternateBidderNotDefined, bidder, alternateBidder)
	}

	if !adapterCfg.Enabled {
		// config has bidder entry but is not enabled, report it
		return false, fmt.Errorf(ErrAlternateBidderDisabled, bidder, alternateBidder)
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
