package doohqty

import (
	"math"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateImpressionValue(t *testing.T) {
	tests := []struct {
		name        string
		value       impressionValue
		expectedErr string
	}{
		{
			name: "valid-unknown-source",
			value: impressionValue{
				Multiplier: 1,
				SourceType: adcom1.MultiplierUnknown,
			},
		},
		{
			name: "valid-measurement-vendor",
			value: impressionValue{
				Multiplier: 1,
				SourceType: adcom1.MultiplierMeasurementVendorProvided,
				Vendor:     "measurement.example",
			},
		},
		{
			name: "zero-multiplier",
			value: impressionValue{
				Multiplier: 0,
			},
			expectedErr: "multiplier must be greater than 0",
		},
		{
			name: "nan-multiplier",
			value: impressionValue{
				Multiplier: math.NaN(),
			},
			expectedErr: "multiplier must be greater than 0",
		},
		{
			name: "infinite-multiplier",
			value: impressionValue{
				Multiplier: math.Inf(1),
			},
			expectedErr: "multiplier must be greater than 0",
		},
		{
			name: "unsupported-source-type",
			value: impressionValue{
				Multiplier: 1,
				SourceType: adcom1.DOOHMultiplierMeasurementSourceType(99),
			},
			expectedErr: "sourcetype 99 is not supported",
		},
		{
			name: "missing-measurement-vendor",
			value: impressionValue{
				Multiplier: 1,
				SourceType: adcom1.MultiplierMeasurementVendorProvided,
			},
			expectedErr: "vendor is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateImpressionValue(test.value)

			if test.expectedErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedErr)
		})
	}
}

func TestApplyQtyValues(t *testing.T) {
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathImpID, Key: "imp-1"}
	values := map[lookupKey]impressionValue{
		lookup: testLookupValue(lookupPathImpID, "imp-1", 9.25),
	}
	assignments := map[int]lookupKey{0: lookup}

	request := newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1", Qty: &openrtb2.Qty{Multiplier: 1.5}})
	applyQtyValues(request, assignments, values, overwritePolicyMissingOnly)

	require.NotNil(t, request.Imp[0].Qty)
	assert.Equal(t, 1.5, request.Imp[0].Qty.Multiplier)

	applyQtyValues(request, assignments, values, overwritePolicyAlways)

	require.NotNil(t, request.Imp[0].Qty)
	assert.Equal(t, 9.25, request.Imp[0].Qty.Multiplier)
	assert.Equal(t, adcom1.MultiplierMeasurementVendorProvided, request.Imp[0].Qty.SourceType)
	assert.Equal(t, "measurement.example", request.Imp[0].Qty.Vendor)
}

func TestHasQtyMutationPredicates(t *testing.T) {
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathImpID, Key: "imp-1"}
	values := map[lookupKey]impressionValue{
		lookup: testLookupValue(lookupPathImpID, "imp-1", 9.25),
	}
	assignments := map[int]lookupKey{0: lookup}
	request := newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1", Qty: &openrtb2.Qty{Multiplier: 1.5}})

	assert.False(t, hasImpressionNeedingQty(request, assignments, overwritePolicyMissingOnly))
	assert.True(t, hasImpressionNeedingQty(request, assignments, overwritePolicyAlways))
	assert.False(t, hasApplicableQtyMutation(request, assignments, values, overwritePolicyMissingOnly))
	assert.True(t, hasApplicableQtyMutation(request, assignments, values, overwritePolicyAlways))
	assert.False(t, hasApplicableQtyMutation(request, assignments, nil, overwritePolicyAlways))
}
