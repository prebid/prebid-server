package vastbidder

import (
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

//TagBidder is default implementation of ITagBidder
type TagBidder struct {
	adapters.Bidder
	bidderName    openrtb_ext.BidderName
	adapterConfig *config.Adapter
}

//MakeRequests will contains default definition for processing queries
func (a *TagBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	bidderMacro := GetNewBidderMacro(a.bidderName)
	bidderMapper := GetDefaultMapper()
	macroProcessor := NewMacroProcessor(bidderMacro, bidderMapper)

	//Setting config parameters
	//bidderMacro.SetBidderConfig(a.bidderConfig)
	bidderMacro.SetAdapterConfig(a.adapterConfig)
	bidderMacro.InitBidRequest(request)

	requestData := []*adapters.RequestData{}
	for impIndex := range request.Imp {
		bidderExt, err := bidderMacro.LoadImpression(&request.Imp[impIndex])
		if nil != err {
			continue
		}

		//iterate each vast tags, and load vast tag
		for vastTagIndex, tag := range bidderExt.Tags {
			//load vasttag
			bidderMacro.LoadVASTTag(tag)

			//Setting Bidder Level Keys
			bidderKeys := bidderMacro.GetBidderKeys()
			macroProcessor.SetBidderKeys(bidderKeys)

			uri := macroProcessor.Process(bidderMacro.GetURI())

			// append custom headers if any
			headers := bidderMacro.getAllHeaders()

			requestData = append(requestData, &adapters.RequestData{
				Params: &adapters.BidRequestParams{
					ImpIndex:     impIndex,
					VASTTagIndex: vastTagIndex,
				},
				Method:  `GET`,
				Uri:     uri,
				Headers: headers,
			})
		}
	}

	return requestData, nil
}

//MakeBids makes bids
func (a *TagBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	//response validation can be done here independently
	//handler, err := GetResponseHandler(a.bidderConfig.ResponseType)
	handler, err := GetResponseHandler(VASTTagHandlerType)
	if nil != err {
		return nil, []error{err}
	}
	return handler.MakeBids(internalRequest, externalRequest, response)
}

//NewTagBidder is an constructor for TagBidder
func NewTagBidder(bidderName openrtb_ext.BidderName, config config.Adapter) *TagBidder {
	obj := &TagBidder{
		bidderName:    bidderName,
		adapterConfig: &config,
	}
	return obj
}

// Builder builds a new instance of the 33Across adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	return NewTagBidder(bidderName, config), nil
}
