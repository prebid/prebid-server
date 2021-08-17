package openrtb2

import (
	"fmt"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func processInterstitials(req *openrtb_ext.RequestWrapper) error {
	unmarshalled := true
	for i := range req.Imp {
		if req.Imp[i].Instl == 1 {
			var prebid *openrtb_ext.ExtDevicePrebid
			if unmarshalled {
				if req.Device.Ext == nil {
					// No special interstitial support requested, so bail as there is nothing to do
					return nil
				}
				deviceExt, err := req.GetDeviceExt()
				if err != nil {
					return err
				}
				prebid = deviceExt.GetPrebid()
				if prebid.Interstitial == nil {
					// No special interstitial support requested, so bail as there is nothing to do
					return nil
				}
			}
			err := processInterstitialsForImp(&req.Imp[i], prebid, req.Device)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func processInterstitialsForImp(imp *openrtb2.Imp, devExtPrebid *openrtb_ext.ExtDevicePrebid, device *openrtb2.Device) error {
	var maxWidth, maxHeight, minWidth, minHeight int64
	if imp.Banner == nil {
		// custom interstitial support is only available for banner requests.
		return nil
	}
	if len(imp.Banner.Format) > 0 {
		maxWidth = imp.Banner.Format[0].W
		maxHeight = imp.Banner.Format[0].H
	}
	if maxWidth < 2 && maxHeight < 2 {
		// This catches size 1x1 as "use device size"
		if device == nil {
			return &errortypes.BadInput{Message: fmt.Sprintf("Unable to read max interstitial size for Imp id=%s (No Device and no Format objects)", imp.ID)}
		}
		maxWidth = device.W
		maxHeight = device.H
	}
	minWidth = (maxWidth * devExtPrebid.Interstitial.MinWidthPerc) / 100
	minHeight = (maxHeight * devExtPrebid.Interstitial.MinHeightPerc) / 100
	imp.Banner.Format = genInterstitialFormat(minWidth, maxWidth, minHeight, maxHeight)
	if len(imp.Banner.Format) == 0 {
		return &errortypes.BadInput{Message: fmt.Sprintf("Unable to set interstitial size list for Imp id=%s (No valid sizes between %dx%d and %dx%d)", imp.ID, minWidth, minHeight, maxWidth, maxHeight)}
	}
	return nil
}

func genInterstitialFormat(minWidth, maxWidth, minHeight, maxHeight int64) []openrtb2.Format {
	sizes := make([]config.InterstitialSize, 0, 10)
	for _, size := range config.ResolvedInterstitialSizes {
		if int64(size.Width) >= minWidth && int64(size.Width) <= maxWidth && int64(size.Height) >= minHeight && int64(size.Height) <= maxHeight {
			sizes = append(sizes, size)
			if len(sizes) >= 10 {
				// we have enough sizes
				break
			}
		}
	}
	formatList := make([]openrtb2.Format, 0, len(sizes))
	for _, size := range sizes {
		formatList = append(formatList, openrtb2.Format{W: int64(size.Width), H: int64(size.Height)})
	}
	return formatList
}
