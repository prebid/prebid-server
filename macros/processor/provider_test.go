package processor

import (
	"reflect"
	"testing"
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
			args: args{keys: []string{BidIDKey, AccountIDKey, AppBundleKey, PubDomainkey,
				PageURLKey, AccountIDKey, LmtTrackingKey, ConsentKey}},
			want: map[string]string{"PBS-ACCOUNTID": "testpublisherID", "PBS-APPBUNDLE": "testdomain", "PBS-BIDID": "bidId123", "PBS-GDPRCONSENT": "yes", "PBS-LIMITADTRACKING": "10", "PBS-PAGEURL": "pageurltest", "PBS-PUBDOMAIN": "publishertestdomain"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macroProvider := NewProvider(req)
			macroProvider.SetContext(bid, nil, "test")
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
			args: args{keys: []string{BidIDKey, AccountIDKey, AppBundleKey, PubDomainkey,
				PageURLKey, AccountIDKey, LmtTrackingKey, ConsentKey}},
			want: map[string]string{"PBS-ACCOUNTID": "testpublisherID", "PBS-APPBUNDLE": "testdomain", "PBS-BIDID": "bidId123", "PBS-GDPRCONSENT": "yes", "PBS-LIMITADTRACKING": "10", "PBS-PAGEURL": "pageurltest", "PBS-PUBDOMAIN": "publishertestdomain"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macroProvider := NewProvider(req)
			macroProvider.SetContext(bid, nil, "test")
			for _, key := range tt.args.keys {
				if got := macroProvider.GetMacro(key); got != tt.want[key] {
					t.Errorf("macroProvider.GetMacro() = %v, want %v", got, tt.want[key])
				}
			}
		})
	}
}
