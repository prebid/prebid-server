package ctv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeImpressionID(t *testing.T) {
	type args struct {
		id string
	}
	type want struct {
		id  string
		seq int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "TC1",
			args: args{id: "impid"},
			want: want{id: "impid", seq: 0},
		},
		{
			name: "TC2",
			args: args{id: "impid_1"},
			want: want{id: "impid", seq: 1},
		},
		{
			name: "TC1",
			args: args{id: "impid_1_2"},
			want: want{id: "impid_1", seq: 2},
		},
		{
			name: "TC1",
			args: args{id: "impid_1_x"},
			want: want{id: "impid_1_x", seq: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, seq := DecodeImpressionID(tt.args.id)
			assert.Equal(t, tt.want.id, id)
			assert.Equal(t, tt.want.seq, seq)
		})
	}
}
