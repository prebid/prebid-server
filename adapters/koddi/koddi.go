package koddi

import (
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type KoddiAdapter struct {
	URI string
	impUrl string;
	clickUrl string;
	conversionUrl string;
}

func (a *KoddiAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return nil, nil
}
func (a *KoddiAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
		return nil, nil
}

// Builder builds a new instance of the Koddi adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &KoddiAdapter{
		URI: config.Endpoint,
	    impUrl: config.ComParams.ImpTracker,
		clickUrl: config.ComParams.ClickTracker,
		conversionUrl: config.ComParams.ConversionTracker,
	}

	return bidder, nil
}

func (a *KoddiAdapter) buildEndpointURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.URI, endpointParams)
}

func (a *KoddiAdapter) buildImpressionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.impUrl, endpointParams)
}

func (a *KoddiAdapter) buildClickURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.clickUrl, endpointParams)
}

func (a *KoddiAdapter) buildConversionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.conversionUrl, endpointParams)
}