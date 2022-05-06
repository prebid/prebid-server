package vastbidder

import (
	"errors"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
)

//ITagRequestHandler parse bidder request
type ITagRequestHandler interface {
	MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error)
}

//ITagResponseHandler parse bidder response
type ITagResponseHandler interface {
	Validate(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) []error
	MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error)
}

//HandlerType list of tag based response handlers
type HandlerType string

const (
	VASTTagHandlerType HandlerType = `vasttag`
)

//GetResponseHandler returns response handler
func GetResponseHandler(responseType HandlerType) (ITagResponseHandler, error) {
	switch responseType {
	case VASTTagHandlerType:
		return NewVASTTagResponseHandler(), nil
	}
	return nil, errors.New(`Unkown Response Handler`)
}

func GetRequestHandler(responseType HandlerType) (ITagRequestHandler, error) {
	switch responseType {
	case VASTTagHandlerType:
		return nil, nil
	}
	return nil, errors.New(`Unkown Response Handler`)
}
