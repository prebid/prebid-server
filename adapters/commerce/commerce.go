package commerce

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type CommerceAdapter struct {
	endpoint *template.Template
	impurl *template.Template
	clickurl *template.Template
	conversionurl *template.Template
}

func (a *CommerceAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	host := "commerce"
	endPoint,_ := a.buildEndpointURL(host)
	errs := make([]error, 0, len(request.Imp))

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     endPoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
	
}
func (a *CommerceAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
		return nil, nil
}

// Builder builds a new instance of the Commerce adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {

	endpointtemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	impurltemplate, err := template.New("impurlTemplate").Parse(config.ComParams.ImpTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse imp url template: %v", err)
	}

	clickurltemplate, err := template.New("clickurlTemplate").Parse(config.ComParams.ClickTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse click url template: %v", err)
	}

	conversionurltemplate, err := template.New("endpointTemplate").Parse(config.ComParams.ConversionTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse conversion url template: %v", err)
	}

	bidder := &CommerceAdapter{
		endpoint: endpointtemplate,
	    impurl: impurltemplate,
		clickurl: clickurltemplate,
		conversionurl: conversionurltemplate,
	}

	return bidder, nil
}

func (a *CommerceAdapter) buildEndpointURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *CommerceAdapter) buildImpressionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.impurl, endpointParams)
}

func (a *CommerceAdapter) buildClickURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.clickurl, endpointParams)
}

func (a *CommerceAdapter) buildConversionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.conversionurl, endpointParams)
}