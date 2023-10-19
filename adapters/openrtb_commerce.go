package adapters

import (
	"encoding/json"
	"errors"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
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

func ValidateCommRequest(request *openrtb2.BidRequest ) (*openrtb_ext.ExtImpCommerce, *openrtb_ext.ExtSiteCommerce,[]error) {
	var commerceExt *openrtb_ext.ExtImpCommerce
	var siteExt *openrtb_ext.ExtSiteCommerce
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

	if len(errors) > 0 {
		return nil, nil, errors
	}

	return commerceExt, siteExt, nil
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
