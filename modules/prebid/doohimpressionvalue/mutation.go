package doohimpressionvalue

import (
	"fmt"
	"math"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func validateImpressionValue(value impressionValue) error {
	if value.Multiplier <= 0 || math.IsNaN(value.Multiplier) || math.IsInf(value.Multiplier, 0) {
		return fmt.Errorf("multiplier must be greater than 0")
	}

	switch value.SourceType {
	case adcom1.MultiplierUnknown,
		adcom1.MultiplierMeasurementVendorProvided,
		adcom1.MultiplierPublisherProvided,
		adcom1.MultiplierExchangeProvided:
	default:
		return fmt.Errorf("sourcetype %d is not supported", value.SourceType)
	}

	if value.SourceType == adcom1.MultiplierMeasurementVendorProvided && value.Vendor == "" {
		return fmt.Errorf("vendor is required when sourcetype is %d", value.SourceType)
	}

	return nil
}

func hasApplicableQtyMutation(request *openrtb_ext.RequestWrapper, assignments map[int]lookupKey, values map[lookupKey]impressionValue, policy overwritePolicy) bool {
	if request == nil {
		return false
	}

	for index, imp := range request.GetImp() {
		lookup, ok := assignments[index]
		if !ok {
			continue
		}
		if _, ok := values[lookup]; !ok {
			continue
		}
		if policy == overwritePolicyAlways || imp.Qty == nil {
			return true
		}
	}

	return false
}

func hasImpressionNeedingQty(request *openrtb_ext.RequestWrapper, assignments map[int]lookupKey, policy overwritePolicy) bool {
	if request == nil {
		return false
	}
	if policy == overwritePolicyAlways {
		return len(assignments) > 0
	}

	for index, imp := range request.GetImp() {
		if _, ok := assignments[index]; ok && imp.Qty == nil {
			return true
		}
	}

	return false
}

func applyQtyValues(request *openrtb_ext.RequestWrapper, assignments map[int]lookupKey, values map[lookupKey]impressionValue, policy overwritePolicy) {
	if request == nil {
		return
	}

	for index, imp := range request.GetImp() {
		lookup, ok := assignments[index]
		if !ok {
			continue
		}

		value, ok := values[lookup]
		if !ok {
			continue
		}

		if policy != overwritePolicyAlways && imp.Qty != nil {
			continue
		}

		imp.Qty = &openrtb2.Qty{
			Multiplier: value.Multiplier,
			SourceType: value.SourceType,
			Vendor:     value.Vendor,
		}
	}
}
