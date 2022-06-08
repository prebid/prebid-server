package errortypes

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	type args struct {
		err error
	}
	type want struct {
		errorMessage string
		code         int
		severity     Severity
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `normal_error`,
			args: args{
				err: fmt.Errorf(`normal_error`),
			},
			want: want{
				errorMessage: `normal_error`,
				code:         UnknownErrorCode,
				severity:     SeverityUnknown,
			},
		},
		{
			name: `Timeout`,
			args: args{
				err: &Timeout{Message: `Timeout_ErrorMessage`},
			},
			want: want{
				errorMessage: `Timeout_ErrorMessage`,
				code:         TimeoutErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `BadInput`,
			args: args{
				err: &BadInput{Message: `BadInput_ErrorMessage`},
			},
			want: want{
				errorMessage: `BadInput_ErrorMessage`,
				code:         BadInputErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `BlacklistedApp`,
			args: args{
				err: &BlacklistedApp{Message: `BlacklistedApp_ErrorMessage`},
			},
			want: want{
				errorMessage: `BlacklistedApp_ErrorMessage`,
				code:         BlacklistedAppErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `BlacklistedAcct`,
			args: args{
				err: &BlacklistedAcct{Message: `BlacklistedAcct_ErrorMessage`},
			},
			want: want{
				errorMessage: `BlacklistedAcct_ErrorMessage`,
				code:         BlacklistedAcctErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `AcctRequired`,
			args: args{
				err: &AcctRequired{Message: `AcctRequired_ErrorMessage`},
			},
			want: want{
				errorMessage: `AcctRequired_ErrorMessage`,
				code:         AcctRequiredErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `BadServerResponse`,
			args: args{
				err: &BadServerResponse{Message: `BadServerResponse_ErrorMessage`},
			},
			want: want{
				errorMessage: `BadServerResponse_ErrorMessage`,
				code:         BadServerResponseErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `FailedToRequestBids`,
			args: args{
				err: &FailedToRequestBids{Message: `FailedToRequestBids_ErrorMessage`},
			},
			want: want{
				errorMessage: `FailedToRequestBids_ErrorMessage`,
				code:         FailedToRequestBidsErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `BidderTemporarilyDisabled`,
			args: args{
				err: &BidderTemporarilyDisabled{Message: `BidderTemporarilyDisabled_ErrorMessage`},
			},
			want: want{
				errorMessage: `BidderTemporarilyDisabled_ErrorMessage`,
				code:         BidderTemporarilyDisabledErrorCode,
				severity:     SeverityWarning,
			},
		},
		{
			name: `Warning`,
			args: args{
				err: &Warning{Message: `Warning_ErrorMessage`, WarningCode: UnknownWarningCode},
			},
			want: want{
				errorMessage: `Warning_ErrorMessage`,
				code:         UnknownWarningCode,
				severity:     SeverityWarning,
			},
		},
		{
			name: `BidderFailedSchemaValidation`,
			args: args{
				err: &BidderFailedSchemaValidation{Message: `BidderFailedSchemaValidation_ErrorMessage`},
			},
			want: want{
				errorMessage: `BidderFailedSchemaValidation_ErrorMessage`,
				code:         BidderFailedSchemaValidationErrorCode,
				severity:     SeverityWarning,
			},
		},
		{
			name: `NoBidPrice`,
			args: args{
				err: &NoBidPrice{Message: `NoBidPrice_ErrorMessage`},
			},
			want: want{
				errorMessage: `NoBidPrice_ErrorMessage`,
				code:         NoBidPriceErrorCode,
				severity:     SeverityWarning,
			},
		},
		{
			name: `AdpodPrefiltering`,
			args: args{
				err: &AdpodPrefiltering{Message: `AdpodPrefiltering_ErrorMessage`},
			},
			want: want{
				errorMessage: `AdpodPrefiltering_ErrorMessage`,
				code:         AdpodPrefilteringErrorCode,
				severity:     SeverityFatal,
			},
		},
		{
			name: `AdpodPostFiltering`,
			args: args{
				err: &AdpodPostFiltering{Message: `AdpodPostFiltering_ErrorMessage`},
			},
			want: want{
				errorMessage: `AdpodPostFiltering_ErrorMessage`,
				code:         AdpodPostFilteringWarningCode,
				severity:     SeverityWarning,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.errorMessage, tt.args.err.Error())
			if code, ok := tt.args.err.(Coder); ok {
				assert.Equal(t, tt.want.code, code.Code())
				assert.Equal(t, tt.want.severity, code.Severity())
			}
		})
	}
}
