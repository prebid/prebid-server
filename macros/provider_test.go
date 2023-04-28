package macros

import (
	"testing"

	"github.com/prebid/prebid-server/exchange/entities"
	"github.com/stretchr/testify/assert"
)

func Test_macroProvider_GetMacro(t *testing.T) {
	type args struct {
		keys []string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "get all macros success",
			args: args{keys: []string{MacroKeyBidID, MacroKeyAccountID, MacroKeyAppBundle, MacroKeyPubDomain,
				MacroKeyPageURL, MacroKeyAccountID, MacroKeyLmtTracking, MacroKeyConsent}},
			want: map[string]string{"PBS-ACCOUNTID": "testpublisherID", "PBS-APPBUNDLE": "testbundle", "PBS-BIDID": "bidId123", "PBS-GDPRCONSENT": "yes", "PBS-LIMITADTRACKING": "10", "PBS-PAGEURL": "pageurltest", "PBS-PUBDOMAIN": "publishertestdomain"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macroProvider := NewProvider(req)

			macroProvider.PopulateBidMacros(&entities.PbsOrtbBid{Bid: bid}, "test")
			macroProvider.PopulateEventMacros("123", "vast", "firstQuartile")
			for _, key := range tt.args.keys {
				got := macroProvider.GetMacro(key)
				assert.Equal(t, tt.want[key], got, tt.name)
			}
		})
	}
}
