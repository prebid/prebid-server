package exchange

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func Test_httpInfoToNonBidReason(t *testing.T) {
	type args struct {
		httpInfo *httpCallInfo
	}
	tests := []struct {
		name string
		args args
		want openrtb_ext.NonBidReason
	}{
		{
			name: "error-timeout",
			args: args{
				httpInfo: &httpCallInfo{
					err: &errortypes.Timeout{},
				},
			},
			want: openrtb_ext.ErrorTimeout,
		},
		{
			name: "error-general",
			args: args{
				httpInfo: &httpCallInfo{
					err: errors.New("some_error"),
				},
			},
			want: openrtb_ext.ErrorGeneral,
		},
		{
			name: "error-bidderUnreachable",
			args: args{
				httpInfo: &httpCallInfo{
					err: syscall.ECONNREFUSED,
				},
			},
			want: openrtb_ext.ErrorBidderUnreachable,
		},
		{
			name: "error-biddersUnreachable-no-such-host",
			args: args{
				httpInfo: &httpCallInfo{
					err: &net.DNSError{IsNotFound: true},
				},
			},
			want: openrtb_ext.ErrorBidderUnreachable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := httpInfoToNonBidReason(tt.args.httpInfo)
			assert.Equal(t, tt.want, actual)
		})
	}
}
