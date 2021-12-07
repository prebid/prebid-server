package lmt

import (
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/iosutil"
)

var (
	int8Zero int8 = 0
	int8One  int8 = 1
)

// ModifyForIOS modifies the request's LMT flag based on iOS version and identity.
func ModifyForIOS(req *openrtb2.BidRequest) {
	modifiers := map[iosutil.VersionClassification]modifier{
		iosutil.Version140:          modifyForIOS14X,
		iosutil.Version141:          modifyForIOS14X,
		iosutil.Version142OrGreater: modifyForIOS142OrGreater,
	}
	modifyForIOS(req, modifiers)
}

func modifyForIOS(req *openrtb2.BidRequest, modifiers map[iosutil.VersionClassification]modifier) {
	if !isRequestForIOS(req) {
		return
	}

	versionClassification := iosutil.DetectVersionClassification(req.Device.OSV)
	if modifier, ok := modifiers[versionClassification]; ok {
		modifier(req)
	}
}

func isRequestForIOS(req *openrtb2.BidRequest) bool {
	return req != nil && req.App != nil && req.Device != nil && strings.EqualFold(req.Device.OS, "ios")
}

type modifier func(req *openrtb2.BidRequest)

func modifyForIOS14X(req *openrtb2.BidRequest) {
	if req.Device.IFA == "" || req.Device.IFA == "00000000-0000-0000-0000-000000000000" {
		req.Device.Lmt = &int8One
	} else {
		req.Device.Lmt = &int8Zero
	}
}

func modifyForIOS142OrGreater(req *openrtb2.BidRequest) {
	atts, err := openrtb_ext.ParseDeviceExtATTS(req.Device.Ext)
	if err != nil || atts == nil {
		return
	}

	switch *atts {
	case openrtb_ext.IOSAppTrackingStatusNotDetermined:
		req.Device.Lmt = &int8Zero
	case openrtb_ext.IOSAppTrackingStatusRestricted:
		req.Device.Lmt = &int8One
	case openrtb_ext.IOSAppTrackingStatusDenied:
		req.Device.Lmt = &int8One
	case openrtb_ext.IOSAppTrackingStatusAuthorized:
		req.Device.Lmt = &int8Zero
	}
}
