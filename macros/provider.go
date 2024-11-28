package macros

import (
	"net/url"
	"strconv"
	"time"

	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const (
	MacroKeyBidID       = "PBS-BIDID"
	MacroKeyAppBundle   = "PBS-APPBUNDLE"
	MacroKeyDomain      = "PBS-DOMAIN"
	MacroKeyPubDomain   = "PBS-PUBDOMAIN"
	MacroKeyPageURL     = "PBS-PAGEURL"
	MacroKeyAccountID   = "PBS-ACCOUNTID"
	MacroKeyLmtTracking = "PBS-LIMITADTRACKING"
	MacroKeyConsent     = "PBS-GDPRCONSENT"
	MacroKeyBidder      = "PBS-BIDDER"
	MacroKeyIntegration = "PBS-INTEGRATION"
	MacroKeyVastCRTID   = "PBS-VASTCRTID"
	MacroKeyTimestamp   = "PBS-TIMESTAMP"
	MacroKeyAuctionID   = "PBS-AUCTIONID"
	MacroKeyChannel     = "PBS-CHANNEL"
	MacroKeyEventType   = "PBS-EVENTTYPE"
	MacroKeyVastEvent   = "PBS-VASTEVENT"
)

const (
	customMacroLength = 100
	CustomMacroPrefix = "PBS-MACRO-"
)

type MacroProvider struct {
	// macros map stores macros key values
	macros map[string]string
}

// NewBuilder returns the instance of macro buidler
func NewProvider(reqWrapper *openrtb_ext.RequestWrapper) *MacroProvider {
	macroProvider := &MacroProvider{macros: map[string]string{}}
	macroProvider.populateRequestMacros(reqWrapper)
	return macroProvider
}

func (b *MacroProvider) populateRequestMacros(reqWrapper *openrtb_ext.RequestWrapper) {
	b.macros[MacroKeyTimestamp] = strconv.Itoa(int(time.Now().Unix()))
	reqExt, err := reqWrapper.GetRequestExt()
	if err == nil && reqExt != nil {
		if reqPrebidExt := reqExt.GetPrebid(); reqPrebidExt != nil {
			for key, value := range reqPrebidExt.Macros {
				customMacroKey := CustomMacroPrefix + key                     // Adding prefix PBS-MACRO to custom macro keys
				b.macros[customMacroKey] = truncate(value, customMacroLength) // limit the custom macro value  to 100 chars only
			}

			if reqPrebidExt.Integration != "" {
				b.macros[MacroKeyIntegration] = reqPrebidExt.Integration
			}

			if reqPrebidExt.Channel != nil {
				b.macros[MacroKeyChannel] = reqPrebidExt.Channel.Name
			}
		}
	}
	b.macros[MacroKeyAuctionID] = reqWrapper.ID
	if reqWrapper.App != nil {
		if reqWrapper.App.Bundle != "" {
			b.macros[MacroKeyAppBundle] = reqWrapper.App.Bundle
		}

		if reqWrapper.App.Domain != "" {
			b.macros[MacroKeyDomain] = reqWrapper.App.Domain
		}

		if reqWrapper.App.Publisher != nil {
			if reqWrapper.App.Publisher.Domain != "" {
				b.macros[MacroKeyPubDomain] = reqWrapper.App.Publisher.Domain
			}
			if reqWrapper.App.Publisher.ID != "" {
				b.macros[MacroKeyAccountID] = reqWrapper.App.Publisher.ID
			}
		}
	}

	if reqWrapper.Site != nil {
		if reqWrapper.Site.Page != "" {
			b.macros[MacroKeyPageURL] = reqWrapper.Site.Page
		}

		if reqWrapper.Site.Domain != "" {
			b.macros[MacroKeyDomain] = reqWrapper.Site.Domain
		}

		if reqWrapper.Site.Publisher != nil {
			if reqWrapper.Site.Publisher.Domain != "" {
				b.macros[MacroKeyPubDomain] = reqWrapper.Site.Publisher.Domain
			}

			if reqWrapper.Site.Publisher.ID != "" {
				b.macros[MacroKeyAccountID] = reqWrapper.Site.Publisher.ID
			}
		}
	}

	if reqWrapper.User != nil && len(reqWrapper.User.Consent) > 0 {
		b.macros[MacroKeyConsent] = reqWrapper.User.Consent
	}
	if reqWrapper.Device != nil && reqWrapper.Device.Lmt != nil {
		b.macros[MacroKeyLmtTracking] = strconv.Itoa(int(*reqWrapper.Device.Lmt))
	}

}

func (b *MacroProvider) GetMacro(key string) string {
	return url.QueryEscape(b.macros[key])
}

func (b *MacroProvider) PopulateBidMacros(bid *entities.PbsOrtbBid, seat string) {
	if bid.Bid != nil {
		if bid.GeneratedBidID != "" {
			b.macros[MacroKeyBidID] = bid.GeneratedBidID
		} else {
			b.macros[MacroKeyBidID] = bid.Bid.ID
		}
	}
	b.macros[MacroKeyBidder] = seat
}

func (b *MacroProvider) PopulateEventMacros(vastCreativeID, eventType, vastEvent string) {
	b.macros[MacroKeyVastCRTID] = vastCreativeID
	b.macros[MacroKeyEventType] = eventType
	b.macros[MacroKeyVastEvent] = vastEvent
}

func truncate(text string, width uint) string {
	r := []rune(text)
	if uint(len(r)) < (width) {
		return text
	}
	trunc := r[:width]
	return string(trunc)
}
