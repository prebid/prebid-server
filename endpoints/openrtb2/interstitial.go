package openrtb2

import (
	"encoding/json"
	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func processInterstitials(req *openrtb.BidRequest) error {
	var devExt openrtb_ext.ExtDevice
	unmarshalled := true
	for i := range req.Imp {
		if req.Imp[i].Instl == 1 {
			if unmarshalled {
				if req.Device.Ext == nil {
					// No special interstitial support requested, so bail as there is nothing to do
					return nil
				}
				err := json.Unmarshal(req.Device.Ext, &devExt)
				if err != nil {
					return err
				}
				if devExt.Prebid.Interstitial == nil {
					// No special interstitial support requested, so bail as there is nothing to do
					return nil
				}
			}
			err := processInterstitialsForImp(&req.Imp[i], &devExt, req.Device)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func processInterstitialsForImp(imp *openrtb.Imp, devExt *openrtb_ext.ExtDevice, device *openrtb.Device) error {
	var maxWidth, maxHeight, minWidth, minHeight uint64
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
	minWidth = (maxWidth * devExt.Prebid.Interstitial.MinWidthPerc) / 100
	minHeight = (maxHeight * devExt.Prebid.Interstitial.MinHeightPerc) / 100
	imp.Banner.Format = genInterstitialFormat(minWidth, maxWidth, minHeight, maxHeight)
	if len(imp.Banner.Format) == 0 {
		return &errortypes.BadInput{Message: fmt.Sprintf("Unable to set interstitial size list for Imp id=%s (No valid sizes between %dx%d and %dx%d)", imp.ID, minWidth, minHeight, maxWidth, maxHeight)}
	}
	return nil
}

func genInterstitialFormat(minWidth, maxWidth, minHeight, maxHeight uint64) []openrtb.Format {
	sizes := make(config.InterstitialSizes, 0, 10)
	for _, size := range config.ResolvedInterstitialSizes {
		if size.Width >= minWidth && size.Width <= maxWidth && size.Height >= minHeight && size.Height <= maxHeight {
			sizes = append(sizes, size)
			if len(sizes) >= 10 {
				// we have enough sizes
				break
			}
		}
	}
	formatList := make([]openrtb.Format, 0, len(sizes))
	for _, size := range sizes {
		formatList = append(formatList, openrtb.Format{W: size.Width, H: size.Height})
	}
	return formatList
}
