package adapters

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func GetImpressionExtComm(imp *openrtb2.Imp) (*openrtb_ext.ExtImpCommerce, error) {
	var commerceExt openrtb_ext.ExtImpCommerce
	if err := json.Unmarshal(imp.Ext, &commerceExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Impression extension not provided or can't be unmarshalled",
		}
	}

	return &commerceExt, nil

}


func GetSiteExtComm(request *openrtb2.BidRequest) (*openrtb_ext.ExtSiteCommerce, error) {
	var siteExt openrtb_ext.ExtSiteCommerce

	if request.Site.Ext != nil {
		if err := json.Unmarshal(request.Site.Ext, &siteExt); err != nil {
			return nil, &errortypes.BadInput{
				Message: "Impression extension not provided or can't be unmarshalled",
			}
		}
	}

	return &siteExt, nil

}

func GetRequestExtComm(request *openrtb2.BidRequest) (*openrtb_ext.ExtOWRequest, error) {
	var requestExt openrtb_ext.ExtOWRequest

	if request.Ext != nil {
		if err := json.Unmarshal(request.Ext, &requestExt); err != nil {
			return nil, &errortypes.BadInput{
				Message: "Impression extension not provided or can't be unmarshalled",
			}
		}
	}

	return &requestExt, nil
}


func GetBidderParamsComm(prebidExt *openrtb_ext.ExtOWRequest) (map[string]interface{},error) {
	var bidderParams map[string]interface{}

	if prebidExt.Prebid.BidderParams != nil {
		if err := json.Unmarshal(prebidExt.Prebid.BidderParams, &bidderParams); err != nil {
			return nil, &errortypes.BadInput{
				Message: "Impression extension not provided or can't be unmarshalled",
			}
		}
	}

	return bidderParams, nil
}

func ValidateCommRequest(request *openrtb2.BidRequest ) (*openrtb_ext.ExtImpCommerce, 
	*openrtb_ext.ExtSiteCommerce, map[string]interface{},[]error) {
	var commerceExt *openrtb_ext.ExtImpCommerce
	var siteExt *openrtb_ext.ExtSiteCommerce
	var requestExt *openrtb_ext.ExtOWRequest
	var bidderParams map[string]interface{}


	var err error
	var errors []error

	if len(request.Imp) > 0 {
		commerceExt, err = GetImpressionExtComm(&(request.Imp[0]))
		if err != nil {
			errors = append(errors, err)
		}
	} else {
		errors = append(errors, &errortypes.BadInput{
			Message: "Missing Imp Object",
		})
	}

	siteExt, err = GetSiteExtComm(request)
	if err != nil {
		errors = append(errors, err)
	}

	requestExt, err = GetRequestExtComm(request)
	if err != nil {
		errors = append(errors, err)
	}

	bidderParams, err = GetBidderParamsComm(requestExt)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, nil, nil, errors
	}

	return commerceExt, siteExt, bidderParams, nil
}

func AddDefaultFieldsComm(bid *openrtb2.Bid) {
	if bid != nil {
		bid.CrID = "DefaultCRID"
	}
}

func GenerateUniqueBidIDComm() string {
	id := uuid.New()
	return id.String()
}

