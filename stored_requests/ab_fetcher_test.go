package stored_requests

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_runABSelectionDistribution(t *testing.T) {
	repeatCount := 100000
	rules := []ABConfig{
		{
			Code:      "main",
			Ratio:     80,
			RequestID: "main-request-id",
		},
		{
			Code:  "test5",
			Ratio: 5,
			ImpIDs: map[string]string{
				"imp-id":   "test5-imp-id",
				"other-id": "test5-other-id",
			},
		},
		{
			Code:      "test15",
			Ratio:     15,
			RequestID: "test15-request-id",
			ImpIDs: map[string]string{
				"imp-id": "test15-imp-id",
			},
		},
	}
	rulesBytes, _ := json.Marshal(rules)
	counts := map[string]int{}
	allowedDeviation := 0.02
	for i := 0; i < repeatCount; i++ {
		abConfig, err := runABSelection(rulesBytes)
		counts[abConfig.Code]++
		assert.NoError(t, err)
	}
	for _, val := range rules {
		got := counts[val.Code]
		expected := int(val.Ratio * float64(repeatCount) / 100)
		deviation := math.Abs(1 - float64(got)/float64(expected))
		assert.True(t, deviation <= allowedDeviation,
			fmt.Sprintf("Case %s: Got %d, wanted %d, deviation %.2f%% > expected %.2f%%", val.Code, got, expected, deviation*100, allowedDeviation*100))
	}
}

func Test_unmarshalABConfig(t *testing.T) {
	type args struct {
		abConfigJSON []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *ABConfig
		wantErr bool
	}{
		{
			name: "code required",
			args: args{
				abConfigJSON: []byte(`{}`),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "one of imp_ids request_ids required",
			args: args{
				abConfigJSON: []byte(`{"code":"test"}`),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "request_id only should work",
			args: args{
				abConfigJSON: []byte(`{"code":"test", "request_id": "req-id"}`),
			},
			want:    &ABConfig{Code: "test", RequestID: "req-id", ImpIDs: map[string]string{}},
			wantErr: false,
		},
		{
			name: "imp_ids only should work",
			args: args{
				abConfigJSON: []byte(`{"code":"test", "imp_ids": {"imp-1": "repl-imp-1"}}`),
			},
			want:    &ABConfig{Code: "test", ImpIDs: map[string]string{"imp-1": "repl-imp-1"}},
			wantErr: false,
		},
		{
			name: "normal use",
			args: args{
				abConfigJSON: []byte(`{"code":"test", "request_id": "req-id", "imp_ids": {"imp-1": "repl-imp-1"}}`),
			},
			want:    &ABConfig{Code: "test", RequestID: "req-id", ImpIDs: map[string]string{"imp-1": "repl-imp-1"}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalABConfig(tt.args.abConfigJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalABConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshalABConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
