package floors

import (
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func getFalse() *bool {
	b := false
	return &b
}

func getTrue() *bool {
	b := true
	return &b
}

func TestShouldEnforceFloors(t *testing.T) {
	type args struct {
		bidRequest        *openrtb2.BidRequest
		floorExt          *openrtb_ext.PriceFloorRules
		configEnforceRate int
		f                 func(int) int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No enfocement of floors when enforcePBS is false",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getFalse(),
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
			name: "No enfocement of floors when enforcePBS is true but enforce rate is low",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getTrue(),
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
			name: "No enfocement of floors when enforcePBS is true but enforce rate is low in incoming request",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS:  getTrue(),
						EnforceRate: 10,
					},
					Skipped: getFalse(),
				},
				configEnforceRate: 100,
				f: func(n int) int {
					return n
				},
			},
			want: false,
		},
		{
			name: "Enfocement of floors when skipped is true, non zero value of bidfloor in imp",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    2.2,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getTrue(),
					},
					Skipped: getTrue(),
				},
				configEnforceRate: 98,
				f: func(n int) int {
					return n - 5
				},
			},
			want: true,
		},
		{
			name: "No enfocement of floors when skipped is true, zero value of bidfloor in imp",
			args: args{
				bidRequest: func() *openrtb2.BidRequest {
					r := openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
							{
								BidFloor:    0,
								BidFloorCur: "USD",
							},
						},
					}
					return &r
				}(),
				floorExt: &openrtb_ext.PriceFloorRules{
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS: getTrue(),
					},
					Skipped: getTrue(),
				},
				configEnforceRate: 98,
				f: func(n int) int {
					return n - 5
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldEnforce(tt.args.bidRequest, tt.args.floorExt, tt.args.configEnforceRate, tt.args.f); got != tt.want {
				t.Errorf("ShouldEnforce() = %v, want %v", got, tt.want)
			}
		})
	}
}
