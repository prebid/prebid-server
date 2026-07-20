package eds

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/stretchr/testify/assert"
)

func TestResolveEds(t *testing.T) {
	tests := []struct {
		name       string
		signal     *openrtb2.BidRequest
		request    *openrtb2.BidRequest
		wantDevice string
		wantApp    string
	}{
		{
			name: "signal_eds_only",
			signal: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Ext: json.RawMessage(`{"eds":{"boottime":1710000000000,"totalmem":8589934592}}`),
				},
				App: &openrtb2.App{
					Ext: json.RawMessage(`{"eds":{"install_time":1710000000001}}`),
				},
			},
			wantDevice: `{"boottime":1710000000000,"totalmem":8589934592}`,
			wantApp:    `{"install_time":1710000000001}`,
		},
		{
			name: "request_eds_only",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Ext: json.RawMessage(`{"eds":{"boottime":1710000000000}}`),
				},
				App: &openrtb2.App{
					Ext: json.RawMessage(`{"eds":{"first_launch_time":1710000000002}}`),
				},
			},
			wantDevice: `{"boottime":1710000000000}`,
			wantApp:    `{"first_launch_time":1710000000002}`,
		},
		{
			name: "signal_takes_priority_over_request",
			signal: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Ext: json.RawMessage(`{"eds":{"boottime":1710000000000}}`),
				},
			},
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Ext: json.RawMessage(`{"eds":{"totalmem":8589934592}}`),
				},
			},
			wantDevice: `{"boottime":1710000000000}`,
		},
		{
			name: "ignores_direct_ext_keys",
			signal: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Ext: json.RawMessage(`{"eds":{"boottime":1710000000000},"boottime":999}`),
				},
			},
			wantDevice: `{"boottime":1710000000000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := ResolveEds(tt.signal, tt.request)

			if tt.wantDevice == "" {
				assert.Empty(t, resolved.Device)
			} else {
				assert.JSONEq(t, tt.wantDevice, string(resolved.Device))
			}

			if tt.wantApp == "" {
				assert.Empty(t, resolved.App)
			} else {
				assert.JSONEq(t, tt.wantApp, string(resolved.App))
			}
		})
	}
}

func TestBuildPubmaticEdsBidderParams(t *testing.T) {
	signal := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			Ext: json.RawMessage(`{"eds":{"boottime":1710000000000}}`),
		},
		App: &openrtb2.App{
			Ext: json.RawMessage(`{"eds":{"install_time":1710000000001}}`),
		},
	}

	injected, resolved, err := BuildPubmaticEdsBidderParams(nil, signal, nil, "pubmatic")
	assert.NoError(t, err)
	assert.False(t, resolved.IsEmpty())

	var params map[string]map[string]json.RawMessage
	assert.NoError(t, json.Unmarshal(injected, &params))

	var edsPayload map[string]json.RawMessage
	assert.NoError(t, json.Unmarshal(params["pubmatic"]["eds"], &edsPayload))
	assert.JSONEq(t, string(resolved.Device), string(edsPayload["device"]))
	assert.JSONEq(t, string(resolved.App), string(edsPayload["app"]))
}

func TestBuildPubmaticEdsBidderParamsAppOnlyFromRequest(t *testing.T) {
	request := &openrtb2.BidRequest{
		App: &openrtb2.App{
			Ext: json.RawMessage(`{"eds":{"install_time":1710000000001}}`),
		},
	}

	baseParams, err := json.Marshal(map[string]map[string]interface{}{
		"pubmatic": {"wiid": "wid-eds-app"},
	})
	assert.NoError(t, err)

	injected, resolved, err := BuildPubmaticEdsBidderParams(baseParams, nil, request, "pubmatic")
	assert.NoError(t, err)
	assert.False(t, resolved.IsEmpty())

	var params map[string]map[string]json.RawMessage
	assert.NoError(t, json.Unmarshal(injected, &params))
	assert.JSONEq(t, `{"app":{"install_time":1710000000001}}`, string(params["pubmatic"]["eds"]))
}

func TestStripFromRequest(t *testing.T) {
	req := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			Ext: json.RawMessage(`{"eds":{"boottime":1710000000000},"atts":1}`),
		},
		App: &openrtb2.App{
			Ext: json.RawMessage(`{"eds":{"install_time":1710000000001},"orientation":1}`),
		},
	}

	StripFromRequest(req)

	assert.JSONEq(t, `{"atts":1}`, string(req.Device.Ext))
	assert.JSONEq(t, `{"orientation":1}`, string(req.App.Ext))
}

func TestStripFromRequestRemovesEmptyExt(t *testing.T) {
	req := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			Ext: json.RawMessage(`{"eds":{"boottime":1710000000000}}`),
		},
		App: &openrtb2.App{
			Ext: json.RawMessage(`{"eds":{"install_time":1710000000001}}`),
		},
	}

	StripFromRequest(req)

	assert.Nil(t, req.Device.Ext)
	assert.Nil(t, req.App.Ext)
}

func TestStripFromDeviceCtx(t *testing.T) {
	dvc := models.DeviceCtx{
		Ext: func() *models.ExtDevice {
			ext := models.NewExtDevice()
			_ = ext.UnmarshalJSON(json.RawMessage(`{"eds":{"boottime":1},"atts":1}`))
			return ext
		}(),
	}

	StripFromDeviceCtx(&dvc)

	out, err := dvc.Ext.MarshalJSON()
	assert.NoError(t, err)
	assert.JSONEq(t, `{"atts":1}`, string(out))
}

func TestStripFromDeviceCtxRemovesEmptyExt(t *testing.T) {
	dvc := models.DeviceCtx{
		Ext: func() *models.ExtDevice {
			ext := models.NewExtDevice()
			_ = ext.UnmarshalJSON(json.RawMessage(`{"eds":{"boottime":1}}`))
			return ext
		}(),
	}

	StripFromDeviceCtx(&dvc)

	assert.True(t, dvc.Ext.IsEmpty())
}
