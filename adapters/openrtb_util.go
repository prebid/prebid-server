package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func ExtractAdapterReqBidderParams(bidRequest *openrtb2.BidRequest) (map[string]json.RawMessage, error) {
	if bidRequest == nil {
		return nil, errors.New("error bidRequest should not be nil")
	}

	reqExt := &openrtb_ext.ExtRequest{}
	if len(bidRequest.Ext) > 0 {
		err := json.Unmarshal(bidRequest.Ext, &reqExt)
		if err != nil {
			return nil, fmt.Errorf("error decoding Request.ext : %s", err.Error())
		}
	}

	if reqExt.Prebid.BidderParams == nil {
		return nil, nil
	}

	var bidderParams map[string]json.RawMessage
	err := json.Unmarshal(reqExt.Prebid.BidderParams, &bidderParams)
	if err != nil {
		return nil, err
	}

	return bidderParams, nil
}

func ExtractReqExtBidderParams(bidReq *openrtb2.BidRequest) (map[string]map[string]json.RawMessage, error) {
	if bidReq == nil {
		return nil, errors.New("error bidRequest should not be nil")
	}

	reqExt := &openrtb_ext.ExtRequest{}
	if len(bidReq.Ext) > 0 {
		err := json.Unmarshal(bidReq.Ext, &reqExt)
		if err != nil {
			return nil, fmt.Errorf("error decoding Request.ext : %s", err.Error())
		}
	}

	if reqExt.Prebid.BidderParams == nil {
		return nil, nil
	}

	var bidderParams map[string]map[string]json.RawMessage
	err := json.Unmarshal(reqExt.Prebid.BidderParams, &bidderParams)
	if err != nil {
		return nil, err
	}

	return bidderParams, nil
}
