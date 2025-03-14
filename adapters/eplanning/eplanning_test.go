package eplanning

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"reflect"
	"testing"
)

func Test_unmarshalSupplyChain(t *testing.T) {
	type args struct {
		ext json.RawMessage
	}
	tests := []struct {
		name    string
		args    args
		want    *openrtb2.SupplyChain
		wantErr bool
	}{
		{
			name: "valid_schain",
			args: args{
				ext: json.RawMessage(`{
					"schain": {
						"ver": "1.0",
						"complete": 1,
						"nodes": [
							{
								"asi": "exchange1.com",
								"sid": "1234",
								"hp": 1,
								"ext": "text"
							}
						]
					}
				}`),
			},
			want: &openrtb2.SupplyChain{
				Ver:      "1.0",
				Complete: 1,
				Nodes: []openrtb2.SupplyChainNode{
					{
						ASI: "exchange1.com",
						SID: "1234",
						HP:  openrtb2.Int8Ptr(1),
						Ext: json.RawMessage(`"text"`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid_schain",
			args: args{
				ext: json.RawMessage(`{
					"schain": {
					"ver": 1.0,
					"complete": "1",
					"nodes": [
					{
						"asi": "exchange1.com",
						"sid": "1234",
						"hp": "1",
						"ext": "text"
					}
					]
				}
				}`),
			},
			want:    nil,
			wantErr: true,
		},
	}
	// invalid_schain: 'ver' should be a string; 'complete' and 'hp' should be integers.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalSupplyChain(tt.args.ext)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalSupplyChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshalSupplyChain() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderEPlanning, config.Adapter{
		Endpoint: "http://rtb.e-planning.net/pbs/1"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	setTesting(bidder)
	adapterstest.RunJSONBidderTest(t, "eplanningtest", bidder)
}

func setTesting(bidder adapters.Bidder) {
	bidderEplanning := bidder.(*EPlanningAdapter)
	bidderEplanning.testing = true
}
