package openrtb_ext

import (
	"testing"

	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestExtRegsDSAClone(t *testing.T) {
	tests := []struct {
		name       string
		extRegsDSA *ExtRegsDSA
	}{
		{
			name:       "nil",
			extRegsDSA: nil,
		},
		{
			name: "required_not_nil",
			extRegsDSA: &ExtRegsDSA{
				Required: ptrutil.ToPtr[int8](1),
			},
		},
		{
			name: "pubrender_not_nil",
			extRegsDSA: &ExtRegsDSA{
				PubRender: ptrutil.ToPtr[int8](1),
			},
		},
		{
			name: "datatopub_not_nil",
			extRegsDSA: &ExtRegsDSA{
				DataToPub: ptrutil.ToPtr[int8](1),
			},
		},
		{
			name: "transparency_empty",
			extRegsDSA: &ExtRegsDSA{
				Transparency: []ExtBidDSATransparency{},
			},
		},
		{
			name: "transparency_with_nil_params",
			extRegsDSA: &ExtRegsDSA{
				Transparency: []ExtBidDSATransparency{
					{
						Domain: "domain1",
						Params: nil,
					},
				},
			},
		},
		{
			name: "transparency_with_params",
			extRegsDSA: &ExtRegsDSA{
				Required:  ptrutil.ToPtr[int8](1),
				PubRender: ptrutil.ToPtr[int8](1),
				DataToPub: ptrutil.ToPtr[int8](1),
				Transparency: []ExtBidDSATransparency{
					{
						Domain: "domain1",
						Params: []int{1, 2, 3},
					},
					{
						Domain: "domain2",
						Params: []int{4, 5, 6},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := tt.extRegsDSA.Clone()
			if tt.extRegsDSA == nil {
				assert.Nil(t, clone)
			} else {
				assert.Equal(t, tt.extRegsDSA, clone)

				if tt.extRegsDSA.Required != nil {
					assert.NotSame(t, tt.extRegsDSA.Required, clone.Required)
				}
				if tt.extRegsDSA.PubRender != nil {
					assert.NotSame(t, tt.extRegsDSA.PubRender, clone.PubRender)
				}
				if tt.extRegsDSA.DataToPub != nil {
					assert.NotSame(t, tt.extRegsDSA.DataToPub, clone.DataToPub)
				}
				if tt.extRegsDSA.Transparency != nil {
					assert.NotSame(t, tt.extRegsDSA.Transparency, clone.Transparency)
				}
			}
		})
	}
}
