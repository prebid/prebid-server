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

func (bidderCodes *ExtAlternateBidderCodes) IsValidBidderCode(bidder, alternateBidder string) (bool, error) {
	if alternateBidder == "" || bidder == alternateBidder {
		return true, nil
	}

	if !bidderCodes.Enabled {
		return false, alternateBidderDisabledError(bidder, alternateBidder)
	}

	if bidderCodes.Bidders == nil {
		return false, alternateBidderNotDefinedError(bidder, alternateBidder)
	}

	adapterCfg, ok := bidderCodes.Bidders[bidder]
	if !ok {
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
