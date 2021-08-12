package sharethrough

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const supplyId = "FGMrCMMc"
const strVersion = 8

// Builder builds a new instance of the Sharethrough adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &SharethroughAdapter{
		AdServer: StrOpenRTBTranslator{
			UriHelper: StrUriHelper{BaseURI: config.Endpoint, Clock: Clock{}},
			Util:      Util{Clock: Clock{}},
			UserAgentParsers: UserAgentParsers{
				ChromeVersion:    regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`),
				ChromeiOSVersion: regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`),
				SafariVersion:    regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`),
			},
		},
	}
	return bidder, nil
}

type SharethroughAdapter struct {
	AdServer StrOpenRTBInterface
}

func (a SharethroughAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var reqs []*adapters.RequestData

	if request.Site == nil {
		return nil, []error{fmt.Errorf("request must include a site; in-app placements are not supported")}
	}
	var domain = Util{}.parseDomain(request.Site.Page)

	for i := 0; i < len(request.Imp); i++ {
		req, err := a.AdServer.requestFromOpenRTB(request.Imp[i], request, domain)

		if err != nil {
			return nil, []error{err}
		}
		reqs = append(reqs, req)
	}

	// We never add to the errs slice (early return), so we just create an empty one to return
	return reqs, []error{}
}

func (a SharethroughAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	return a.AdServer.responseToOpenRTB(response.Body, externalRequest)
}
