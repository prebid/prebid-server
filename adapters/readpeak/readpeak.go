package readpeak

import (
	"encoding/json"
	"fmt"
	"net/http"
  
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
  )
  
type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Readpeak adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
	  endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	requestCopy := *request
	requestCopy.id = requestCopy[0].bidderRequestId
	var rpExt openrtb_ext.ImpExtReadpeak
	for i := 0; i < len(requestCopy.Imp); i++ {		
		var impExt adapters.ExtImpBidder
		err := json.Unmarshal(jsonData, &impExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		err := json.Unmarshal(impExt.Bidder, &rpExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		imp := requestCopy.Imp[i]
		if rpExt.TagId != "" {
			imp.tagid = rpExt.TagId
		}
		if rpExt.Bidfloor != 0 {
			imp.bidfloor = rpExt.Bidfloor
		}
		requestCopy.Imp[i] = imp
	}

	if requestCopy.Site != nil {
		site := *requestCopy.Site
		if rpExt.SiteId != "" {
			site.Id = rpExt.SiteId
		}
		if rpExt.PublisherId != "" {
			site.publisher.id = rpExt.PublisherId
		}
		requestCopy.Site = site
	} else if requestCopy.App != nil {
		app := *requestCopy.App
		if rpExt.PublisherId != "" {
			app.publisher.id = rpExt.PublisherId
		}
		requestCopy.App = app
	}
	
	requestJSON, err := json.Marshal(requestCopy)
	if err != nil {
	  return nil, []error{err}
	}
  
	requestData := &adapters.RequestData{
	  Method:  "POST",
	  Uri:     a.endpoint,
	  Body:    requestJSON,
	}
  
	return []*adapters.RequestData{requestData}, errors
}
