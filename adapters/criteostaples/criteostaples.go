package criteostaples

import (
	"fmt"
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type CriteoStaplesAdapter struct {
	endpoint      string
	impurl        *template.Template
	clickurl      *template.Template
	conversionurl *template.Template
}

// Builder builds a new instance of the AdButtler adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {

	impurltemplate, err := template.New("impurlTemplate").Parse(config.ComParams.ImpTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse impression url template: %v", err)
	}

	clickurltemplate, err := template.New("clickurlTemplate").Parse(config.ComParams.ClickTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse click url template: %v", err)
	}

	conversionurltemplate, err := template.New("endpointTemplate").Parse(config.ComParams.ConversionTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse conversion url template: %v", err)
	}

	bidder := &CriteoStaplesAdapter{
		endpoint:      config.Endpoint,
		impurl:        impurltemplate,
		clickurl:      clickurltemplate,
		conversionurl: conversionurltemplate,
	}

	return bidder, nil
}