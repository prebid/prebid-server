package processor

import (
	"net/url"
	"strconv"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	BidIDKey          = "PBS_BIDID"
	AppBundleKey      = "PBS_APPBUNDLE"
	DomainKey         = "PBS_APPBUNDLE"
	PubDomainkey      = "PBS_PUBDOMAIN"
	PageURLKey        = "PBS_PAGEURL"
	AccountIDKey      = "PBS_ACCOUNTID"
	LmtTrackingKey    = "PBS_LIMITADTRACKING"
	ConsentKey        = "PBS_GDPRCONSENT"
	CustomMacroPrefix = "PBS_MACRO_"
	BidderKey         = "##PBS-BIDDER##"
	IntegrationKey    = "##PBS-INTEGRATION##"
)

var (
	bidLevelKeys = []string{BidIDKey, BidderKey}
)

type Provider interface {
	// GetMacro returns the macro value for the given macro key
	GetMacro(key string) string
	// GetAllMacros return all the macros
	GetAllMacros(keys []string) map[string]string
	// SetContext set the bid and imp for the current provider
	SetContext(bid *openrtb2.Bid, imp *openrtb2.Imp, seat string)
}

type macroProvider struct {
	// macros map stores macros key values
	macros map[string]string
}

// NewBuilder returns the instance of macro buidler
func NewProvider(reqWrapper *openrtb_ext.RequestWrapper) Provider {

	macroProvider := &macroProvider{macros: map[string]string{}}
	macroProvider.populateRequestMacros(reqWrapper)
	return macroProvider
}

func (b *macroProvider) populateRequestMacros(reqWrapper *openrtb_ext.RequestWrapper) {
	reqExt, _ := reqWrapper.GetRequestExt()
	if reqExt != nil && reqExt.GetPrebid() != nil {
		for key, value := range reqExt.GetPrebid().Macros {
			customMacroKey := CustomMacroPrefix + key       // Adding prefix PBS_MACRO to custom macro keys
			b.macros[customMacroKey] = truncate(value, 100) // limit the custom macro value  to 100 chars only
		}

		b.macros[IntegrationKey] = reqExt.GetPrebid().Integration
	}

	if reqWrapper.App != nil && reqWrapper.App.Bundle != "" {
		b.macros[AppBundleKey] = reqWrapper.App.Bundle
	}

	if reqWrapper.App != nil && reqWrapper.App.Domain != "" {
		b.macros[DomainKey] = reqWrapper.App.Domain
	}

	if reqWrapper.Site != nil && reqWrapper.Site.Domain != "" {
		b.macros[DomainKey] = reqWrapper.Site.Domain
	}

	if reqWrapper.Site != nil && reqWrapper.Site.Publisher != nil && reqWrapper.Site.Publisher.Domain != "" {
		b.macros[PubDomainkey] = reqWrapper.Site.Publisher.Domain
	}

	if reqWrapper.App != nil && reqWrapper.App.Publisher != nil && reqWrapper.App.Publisher.Domain != "" {
		b.macros[PubDomainkey] = reqWrapper.App.Publisher.Domain
	}

	if reqWrapper.Site != nil {
		b.macros[PageURLKey] = reqWrapper.Site.Page
	}
	userExt, _ := reqWrapper.GetUserExt()
	if userExt != nil && userExt.GetConsent() != nil {
		b.macros[ConsentKey] = *userExt.GetConsent()
	}
	if reqWrapper.Device != nil && reqWrapper.Device.Lmt != nil {
		b.macros[LmtTrackingKey] = strconv.Itoa(int(*reqWrapper.Device.Lmt))
	}

	b.macros[AccountIDKey] = reqWrapper.ID
	if reqWrapper.Site != nil && reqWrapper.Site.Publisher != nil && reqWrapper.Site.Publisher.ID != "" {
		b.macros[AccountIDKey] = reqWrapper.Site.Publisher.ID
	}

	if reqWrapper.App != nil && reqWrapper.App.Publisher != nil && reqWrapper.App.Publisher.ID != "" {
		b.macros[AccountIDKey] = reqWrapper.App.Publisher.ID
	}
}

func (b *macroProvider) GetMacro(key string) string {
	return url.QueryEscape(b.macros[key])
}
func (b *macroProvider) GetAllMacros(keys []string) map[string]string {
	macroValues := map[string]string{}

	for _, key := range keys {
		macroValues[key] = url.QueryEscape(b.macros[key]) // encoding the macro values
	}
	return macroValues
}
func (b *macroProvider) SetContext(bid *openrtb2.Bid, imp *openrtb2.Imp, seat string) {
	b.resetcontext()
	b.macros[BidIDKey] = bid.ID
	b.macros[BidderKey] = seat
}
func (b *macroProvider) resetcontext() {
	for _, key := range bidLevelKeys {
		delete(b.macros, key)
	}
}

func truncate(text string, width int) string {
	if width < 0 {
		return text
	}

	r := []rune(text)
	if len(r) < width {
		return text
	}
	trunc := r[:width]
	return string(trunc)
}
