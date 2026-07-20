package openwrap

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/eds"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestPopulateDeviceExt(t *testing.T) {
	type args struct {
		device *openrtb2.Device
	}

	type want struct {
		deviceCtx models.DeviceCtx
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `nil_request`,
			args: args{},
			want: want{
				deviceCtx: models.DeviceCtx{},
			},
		},
		{
			name: `invalid_device_ext`,
			args: args{
				device: &openrtb2.Device{
					Ext: json.RawMessage(`invalid ext`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{},
			},
		},
		{
			name: `ifa_present`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
				},
			},
		},
		{
			name: `ifa_type_key_absent`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
					Ext: json.RawMessage(`{"anykey":"anyval"}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.UnmarshalJSON([]byte(`{"anykey": "anyval"}`))
						return deviceExt
					}(),
				},
			},
		},
		{
			name: `invalid_data_type_for_ifa_type_key`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
					Ext: json.RawMessage(`{"ifa_type": 123}`)},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
					Ext:       models.NewExtDevice(),
				},
			},
		},
		{
			name: `ifa_type_missing_in_DeviceIFATypeID_mapping`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
					Ext: json.RawMessage(`{"ifa_type": "anything"}`),
				},
			},
			want: want{
				/* removed_invalid_ifatype */
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
					Ext:       models.NewExtDevice(),
				},
			},
		},
		{
			name: `case_insensitive_ifa_type`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
					Ext: json.RawMessage(`{"ifa_type": "DpId"}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
					IFATypeID: ptrutil.ToPtr(models.DeviceIFATypeID[models.DeviceIFATypeDPID]),
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetIFAType("DpId")
						return deviceExt
					}(),
				},
			},
		},
		{
			name: `valid_ifa_type`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
					Ext: json.RawMessage(`{"ifa_type": "sessionid"}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
					IFATypeID: ptrutil.ToPtr(models.DeviceIFATypeID[models.DeviceIFATypeSESSIONID]),
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetIFAType("sessionid")
						return deviceExt
					}(),
				},
			},
		},
		{
			name: `valid_ifa_type_missing_device_ifa`,
			args: args{
				device: &openrtb2.Device{
					Ext: json.RawMessage(`{"ifa_type": "sessionid"}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					Ext: models.NewExtDevice(),
				},
			},
		},
		{
			name: `invalid_device.ext.atts`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
					Ext: json.RawMessage(`{"atts": "invalid_value"}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.UnmarshalJSON([]byte(`{"atts":"invalid_value"}`))
						return deviceExt
					}(),
				},
			},
		},
		{
			name: `valid_device.ext.atts`,
			args: args{
				device: &openrtb2.Device{
					Ext: json.RawMessage(`{"atts": 1}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.UnmarshalJSON([]byte(`{"atts":1}`))
						return deviceExt
					}(),
				},
			},
		},
		{
			name: `deviceID_as_sessionid_incase_ifa_not_present`,
			args: args{
				device: &openrtb2.Device{
					Model: "iphone,11",
					Ext:   json.RawMessage(`{"atts": 1,"session_id": "sample_session_id"}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `sample_session_id`,
					ID:        "sample_session_id",
					Model:     "iphone,11",
					IFATypeID: ptrutil.ToPtr(models.DeviceIFATypeID[models.DeviceIFATypeSESSIONID]),
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.UnmarshalJSON([]byte(`{"atts":1,"session_id": "sample_session_id"}`))
						deviceExt.SetIFAType("sessionid")
						return deviceExt
					}(),
				},
			},
		},
		{
			name: `all_valid_ext_parameters`,
			args: args{
				device: &openrtb2.Device{
					IFA: `test_ifa`,
					Ext: json.RawMessage(`{"ifa_type": "sessionid","atts": 1}`),
				},
			},
			want: want{
				deviceCtx: models.DeviceCtx{
					DeviceIFA: `test_ifa`,
					ID:        "test_ifa",
					IFATypeID: ptrutil.ToPtr(models.DeviceIFATypeID[models.DeviceIFATypeSESSIONID]),
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.UnmarshalJSON([]byte(`{"atts":1}`))
						deviceExt.SetIFAType("sessionid")
						return deviceExt
					}(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dvc := models.DeviceCtx{}
			populateDeviceContext(&dvc, tt.args.device)
			assert.Equal(t, tt.want.deviceCtx, dvc)
		})
	}
}

func TestUpdateDeviceIFADetails(t *testing.T) {
	type args struct {
		dvc *models.DeviceCtx
	}
	tests := []struct {
		name string
		args args
		want *models.DeviceCtx
	}{
		{
			name: `empty`,
			args: args{},
			want: nil,
		},
		{
			name: `device_ext_nil`,
			args: args{
				dvc: &models.DeviceCtx{},
			},
			want: &models.DeviceCtx{},
		},
		{
			name: `device_ext_nil`,
			args: args{
				dvc: &models.DeviceCtx{},
			},
			want: &models.DeviceCtx{},
		},
		{
			name: `ifa_type_missing`,
			args: args{
				dvc: &models.DeviceCtx{
					Ext: &models.ExtDevice{},
				},
			},
			want: &models.DeviceCtx{
				Ext: &models.ExtDevice{},
			},
		},
		{
			name: `ifa_type_present_ifa_missing`,
			args: args{
				dvc: &models.DeviceCtx{
					Ext: func() *models.ExtDevice {
						ext := &models.ExtDevice{}
						ext.SetIFAType(models.DeviceIFATypeDPID)
						return ext
					}(),
				},
			},
			want: &models.DeviceCtx{
				Ext: models.NewExtDevice(),
			},
		},
		{
			name: `wrong_ifa_type`,
			args: args{
				dvc: &models.DeviceCtx{
					DeviceIFA: `sample_ifa_value`,
					Ext: func() *models.ExtDevice {
						ext := &models.ExtDevice{}
						ext.SetIFAType("wrong_ifa_type")
						return ext
					}(),
				},
			},
			want: &models.DeviceCtx{
				DeviceIFA: `sample_ifa_value`,
				Ext:       models.NewExtDevice(),
			},
		},
		{
			name: `valid_ifa_type`,
			args: args{
				dvc: &models.DeviceCtx{
					DeviceIFA: `sample_ifa_value`,
					Ext: func() *models.ExtDevice {
						ext := &models.ExtDevice{}
						ext.SetIFAType(models.DeviceIFATypeDPID)
						return ext
					}(),
				},
			},
			want: &models.DeviceCtx{
				DeviceIFA: `sample_ifa_value`,
				IFATypeID: ptrutil.ToPtr(models.DeviceIFATypeID[models.DeviceIFATypeDPID]),
				Ext: func() *models.ExtDevice {
					ext := &models.ExtDevice{}
					ext.SetIFAType(models.DeviceIFATypeDPID)
					return ext
				}(),
			},
		},
		{
			name: `case_insensitive_ifa_type`,
			args: args{
				dvc: &models.DeviceCtx{
					DeviceIFA: `sample_ifa_value`,
					Ext: func() *models.ExtDevice {
						ext := &models.ExtDevice{}
						ext.SetIFAType(strings.ToUpper(models.DeviceIFATypeDPID))
						return ext
					}(),
				},
			},
			want: &models.DeviceCtx{
				DeviceIFA: `sample_ifa_value`,
				IFATypeID: ptrutil.ToPtr(models.DeviceIFATypeID[models.DeviceIFATypeDPID]),
				Ext: func() *models.ExtDevice {
					ext := &models.ExtDevice{}
					ext.SetIFAType(strings.ToUpper(models.DeviceIFATypeDPID))
					return ext
				}(),
			},
		},
		{
			name: `ifa_type_present_session_id_present`,
			args: args{
				dvc: &models.DeviceCtx{
					Ext: func() *models.ExtDevice {
						ext := &models.ExtDevice{}
						ext.SetIFAType(strings.ToUpper(models.DeviceIFATypeDPID))
						ext.SetSessionID(`sample_session_id`)
						return ext
					}(),
				},
			},
			want: &models.DeviceCtx{
				DeviceIFA: `sample_session_id`,
				IFATypeID: ptrutil.ToPtr(models.DeviceIFATypeID[models.DeviceIFATypeSESSIONID]),
				Ext: func() *models.ExtDevice {
					ext := &models.ExtDevice{}
					ext.SetIFAType(models.DeviceIFATypeSESSIONID)
					ext.SetSessionID(`sample_session_id`)
					return ext
				}(),
			},
		},
		{
			name: `ifa_type_present_session_id_missing`,
			args: args{
				dvc: &models.DeviceCtx{
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetIFAType(models.DeviceIFATypeDPID)
						return deviceExt
					}(),
				},
			},
			want: &models.DeviceCtx{
				Ext: models.NewExtDevice(),
			},
		},
		{
			name: `ifa_type_missing_session_id_present`,
			args: args{
				dvc: &models.DeviceCtx{
					DeviceIFA: `existing_ifa_id`,
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetSessionID(`sample_session_id`)
						return deviceExt
					}(),
				},
			},
			want: &models.DeviceCtx{
				DeviceIFA: `existing_ifa_id`,
				Ext: func() *models.ExtDevice {
					deviceExt := &models.ExtDevice{}
					deviceExt.SetSessionID(`sample_session_id`)
					return deviceExt
				}(),
			},
		},
		{
			name: `ifa_type_missing_ifa_empty_session_id_present`,
			args: args{
				dvc: &models.DeviceCtx{
					DeviceIFA: "",
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetSessionID(`sample_session_id`)
						return deviceExt
					}(),
				},
			},
			want: &models.DeviceCtx{
				IFATypeID: ptrutil.ToPtr(9),
				DeviceIFA: `sample_session_id`,
				Ext: func() *models.ExtDevice {
					deviceExt := &models.ExtDevice{}
					deviceExt.SetIFAType(models.DeviceIFATypeSESSIONID)
					deviceExt.SetSessionID(`sample_session_id`)
					return deviceExt
				}(),
			},
		},
		{
			name: `ifa_type_missing_ifa_not_present_session_id_present`,
			args: args{
				dvc: &models.DeviceCtx{
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetSessionID(`sample_session_id`)
						return deviceExt
					}(),
				},
			},
			want: &models.DeviceCtx{
				IFATypeID: ptrutil.ToPtr(9),
				DeviceIFA: `sample_session_id`,
				Ext: func() *models.ExtDevice {
					deviceExt := &models.ExtDevice{}
					deviceExt.SetIFAType(models.DeviceIFATypeSESSIONID)
					deviceExt.SetSessionID(`sample_session_id`)
					return deviceExt
				}(),
			},
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateDeviceIFADetails(tt.args.dvc)
			assert.Equal(t, tt.want, tt.args.dvc)
		})
	}
}

func TestAmendDeviceObject(t *testing.T) {
	type args struct {
		device *openrtb2.Device
		dvc    *models.DeviceCtx
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.Device
	}{
		{
			name: `any_empty`,
			args: args{},
			want: nil,
		},
		{
			name: `update_ifa_details_1`,
			args: args{
				device: &openrtb2.Device{
					UA: `sample_ua`,
				},
				dvc: &models.DeviceCtx{
					DeviceIFA: `new_ifa`,
				},
			},
			want: &openrtb2.Device{
				UA:  `sample_ua`,
				IFA: `new_ifa`,
			},
		},
		{
			name: `update_ifa_details_2`,
			args: args{
				device: &openrtb2.Device{
					UA:  `sample_ua`,
					IFA: `old_ifa`,
				},
				dvc: &models.DeviceCtx{
					DeviceIFA: `new_ifa`,
				},
			},
			want: &openrtb2.Device{
				UA:  `sample_ua`,
				IFA: `new_ifa`,
			},
		},
		{
			name: `update_ext_1`,
			args: args{
				device: &openrtb2.Device{
					UA:  `sample_ua`,
					IFA: `old_ifa`,
				},
				dvc: &models.DeviceCtx{
					DeviceIFA: `new_ifa`,
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetSessionID("sample_session")
						return deviceExt
					}(),
				},
			},
			want: &openrtb2.Device{
				UA:  `sample_ua`,
				IFA: `new_ifa`,
				Ext: json.RawMessage(`{"session_id":"sample_session"}`),
			},
		},
		{
			name: `update_ext_2`,
			args: args{
				device: &openrtb2.Device{
					UA:  `sample_ua`,
					IFA: `old_ifa`,
					Ext: json.RawMessage(`{"extra_key":"missing"}`),
				},
				dvc: &models.DeviceCtx{
					DeviceIFA: `new_ifa`,
					Ext: func() *models.ExtDevice {
						deviceExt := &models.ExtDevice{}
						deviceExt.SetSessionID("sample_session")
						return deviceExt
					}(),
				},
			},
			want: &openrtb2.Device{
				UA:  `sample_ua`,
				IFA: `new_ifa`,
				Ext: json.RawMessage(`{"session_id":"sample_session"}`),
			},
		},
		{
			name: `clear_ext_when_processed_ext_is_empty`,
			args: args{
				device: &openrtb2.Device{
					Ext: json.RawMessage(`{"ifa_type":"sessionid"}`),
				},
				dvc: &models.DeviceCtx{
					Ext: models.NewExtDevice(),
				},
			},
			want: &openrtb2.Device{},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amendDeviceObject(tt.args.device, tt.args.dvc)
			assert.Equal(t, tt.args.device, tt.want, "mismatched device object")
		})
	}
}

func TestGetDeviceID(t *testing.T) {
	tests := []struct {
		name     string
		dvc      *models.DeviceCtx
		device   *openrtb2.Device
		expected string
	}{
		{
			name:     "Empty input",
			dvc:      &models.DeviceCtx{},
			device:   &openrtb2.Device{},
			expected: "",
		},
		{
			name:     "DeviceIFA present",
			dvc:      &models.DeviceCtx{DeviceIFA: "test-ifa"},
			device:   &openrtb2.Device{IFA: "test-ifa"},
			expected: "test-ifa",
		},
		{
			name:     "DIDSHA1 present",
			dvc:      &models.DeviceCtx{},
			device:   &openrtb2.Device{DIDSHA1: "test-didsha1"},
			expected: "test-didsha1",
		},
		{
			name:     "DIDMD5 present",
			dvc:      &models.DeviceCtx{},
			device:   &openrtb2.Device{DIDMD5: "test-didmd5"},
			expected: "test-didmd5",
		},
		{
			name:     "DPIDSHA1 present",
			dvc:      &models.DeviceCtx{},
			device:   &openrtb2.Device{DPIDSHA1: "test-dpidsha1"},
			expected: "test-dpidsha1",
		},
		{
			name:     "DPIDMD5 present",
			dvc:      &models.DeviceCtx{},
			device:   &openrtb2.Device{DPIDMD5: "test-dpidmd5"},
			expected: "test-dpidmd5",
		},
		{
			name:     "MACSHA1 present",
			dvc:      &models.DeviceCtx{},
			device:   &openrtb2.Device{MACSHA1: "test-macsha1"},
			expected: "test-macsha1",
		},
		{
			name:     "MACMD5 present",
			dvc:      &models.DeviceCtx{},
			device:   &openrtb2.Device{MACMD5: "test-macmd5"},
			expected: "test-macmd5",
		},
		{
			name: "Multiple fields present, DeviceIFA takes precedence",
			dvc:  &models.DeviceCtx{DeviceIFA: "test-ifa"},
			device: &openrtb2.Device{
				IFA:      "test-ifa",
				DIDSHA1:  "test-didsha1",
				DIDMD5:   "test-didmd5",
				DPIDSHA1: "test-dpidsha1",
				DPIDMD5:  "test-dpidmd5",
				MACSHA1:  "test-macsha1",
				MACMD5:   "test-macmd5",
			},
			expected: "test-ifa",
		},
		{
			name: "Multiple fields present, precedence order respected",
			dvc:  &models.DeviceCtx{},
			device: &openrtb2.Device{
				DIDMD5:   "test-didmd5",
				DPIDSHA1: "test-dpidsha1",
				DPIDMD5:  "test-dpidmd5",
				MACSHA1:  "test-macsha1",
				MACMD5:   "test-macmd5",
			},
			expected: "test-didmd5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDeviceID(tt.dvc, tt.device)
			if got != tt.expected {
				t.Errorf("getDeviceID() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDeviceExtClearedAfterIfaTypeProcessing(t *testing.T) {
	device := &openrtb2.Device{
		Ext: json.RawMessage(`{"ifa_type":"sessionid"}`),
	}
	dvc := models.DeviceCtx{}

	populateDeviceContext(&dvc, device)
	eds.StripFromDeviceCtx(&dvc)
	amendDeviceObject(device, &dvc)

	assert.Nil(t, device.Ext)
	assert.True(t, dvc.Ext.IsEmpty())
}
