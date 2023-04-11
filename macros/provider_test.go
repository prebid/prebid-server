package macros

import (
	"reflect"
	"testing"

	"github.com/prebid/prebid-server/exchange/entities"
)

func Test_macroProvider_GetAllMacros(t *testing.T) {

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

			macroProvider.SetContext(MacroContext{
				Bid:            &entities.PbsOrtbBid{Bid: bid},
				Imp:            nil,
				Seat:           "test",
				VastCreativeID: "123",
				VastEventType:  "firstQuartile",
				EventElement:   "tracking",
			})
			if got := macroProvider.GetAllMacros(tt.args.keys); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("macroProvider.GetAllMacros() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

			macroProvider.SetContext(MacroContext{
				Bid:            &entities.PbsOrtbBid{Bid: bid},
				Imp:            nil,
				Seat:           "test",
				VastCreativeID: "123",
				VastEventType:  "firstQuartile",
				EventElement:   "tracking",
			})
			for _, key := range tt.args.keys {
				if got := macroProvider.GetMacro(key); got != tt.want[key] {
					t.Errorf("macroProvider.GetMacro() = %v, want %v", got, tt.want[key])
				}
			}
		})
	}
}
