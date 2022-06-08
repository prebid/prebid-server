package floors

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func getFalse() *bool {
	b := false
	return &b
}

func TestShouldEnforceFloors(t *testing.T) {
	type args struct {
		requestExt        *openrtb_ext.PriceFloorRules
		configEnforceRate int
		f                 func(int) int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No enfocement of floors",
			args: args{
				requestExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: false,
					},
					Skipped: getFalse(),
				},
				configEnforceRate: 10,
				f: func(n int) int {
					return n
				},
			},
			want: false,
		},
		{
			name: "enfocement of floors",
			args: args{
				requestExt: &openrtb_ext.PriceFloorRules{
					Skipped: getFalse(),
				},
				configEnforceRate: 98,
				f: func(n int) int {
					return n - 5
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldEnforceFloors(tt.args.requestExt, tt.args.configEnforceRate, tt.args.f); got != tt.want {
				t.Errorf("ShouldEnforceFloors() = %v, want %v", got, tt.want)
			}
		})
	}
}
