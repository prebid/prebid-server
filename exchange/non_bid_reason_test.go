package exchange

import (
	"errors"
	"reflect"
	"syscall"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v2/errortypes"
)

func Test_httpInfoToNonBidReason(t *testing.T) {
	type args struct {
		httpInfo *httpCallInfo
	}
	tests := []struct {
		name string
		args args
		want openrtb3.NoBidReason
	}{
		{
			name: "Test-ErrorTimeout",
			args: args{
				httpInfo: &httpCallInfo{
					err: &errortypes.Timeout{},
				},
			},
			want: ErrorTimeout,
		},
		{
			name: "Test-ErrorGeneral",
			args: args{
				httpInfo: &httpCallInfo{
					err: errors.New("some_error"),
				},
			},
			want: ErrorGeneral,
		},
		{
			name: "Test-ErrorBidderUnreachable",
			args: args{
				httpInfo: &httpCallInfo{
					err: syscall.ECONNREFUSED,
				},
			},
			want: ErrorBidderUnreachable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := httpInfoToNonBidReason(tt.args.httpInfo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("httpInfoToNonBidReason() = %v, want %v", got, tt.want)
			}
		})
	}
}
