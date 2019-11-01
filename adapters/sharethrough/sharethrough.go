package sharethrough

import (
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"net/http"
	"regexp"
)

const supplyId = "FGMrCMMc"
const strVersion = 7

func NewSharethroughBidder(endpoint string) *SharethroughAdapter {
	return &SharethroughAdapter{
		AdServer: StrOpenRTBTranslator{
			UriHelper: StrUriHelper{BaseURI: endpoint, Clock: Clock{}},
			Util:      Util{Clock: Clock{}},
			UserAgentParsers: UserAgentParsers{
				ChromeVersion:    regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`),
				ChromeiOSVersion: regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`),
				SafariVersion:    regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`),
			},
		},
	}
}

type SharethroughAdapter struct {
	AdServer StrOpenRTBInterface
}

func (a SharethroughAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

func (a SharethroughAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	data, err := getResponseData(response)
	if err != nil {
		return nil, []error{fmt.Errorf("unable to read response body"), err}
	}

	return a.AdServer.responseToOpenRTB(data, externalRequest)
}

func getResponseData(response *adapters.ResponseData) (data []byte, err error) {
	err = nil
	if response.Headers.Get("Content-Encoding") == "gzip" {
		data, err = Util{}.gUnzipData(response.Body)
	} else {
		data = response.Body
	}

	return
}
